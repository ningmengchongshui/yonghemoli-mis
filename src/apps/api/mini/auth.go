package miniapi

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"yonghemolimis/src/settings"
)

type miniSession struct {
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type wechatSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type wechatPhone struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

type douyinSession struct {
	ErrNo   int    `json:"err_no"`
	ErrTips string `json:"err_tips"`
	Data    struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"`
	} `json:"data"`
}

type douyinPhone struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	Watermark       struct {
		AppID string `json:"appid"`
	} `json:"watermark"`
}

type douyinPendingSession struct {
	OpenID     string
	SessionKey string
	ExpiresAt  time.Time
}

type alipayOAuthSession struct {
	UserID string `json:"user_id"`
	Code   string `json:"code"`
	Msg    string `json:"msg"`
}

type douyinClientTokenCache struct {
	Token     string
	ExpiresAt time.Time
}

var miniTokens = struct {
	sync.RWMutex
	store map[string]miniSession
}{store: map[string]miniSession{}}

var douyinPendingTokens = struct {
	sync.RWMutex
	store map[string]douyinPendingSession
}{store: map[string]douyinPendingSession{}}

var douyinClientTokens = struct {
	sync.Mutex
	value douyinClientTokenCache
}{}

func createMiniToken(userID string) string {
	token := randomHex(32)
	miniTokens.Lock()
	defer miniTokens.Unlock()
	miniTokens.store[token] = miniSession{
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	return token
}

func miniUserIDFromToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	miniTokens.RLock()
	sess, ok := miniTokens.store[token]
	miniTokens.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		if ok {
			miniTokens.Lock()
			delete(miniTokens.store, token)
			miniTokens.Unlock()
		}
		return ""
	}
	return sess.UserID
}

func randomHex(size int) string {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func wxCodeToSession(code string) (*wechatSession, error) {
	conf := settings.Conf.MiniWechat
	if conf == nil || conf.AppID == "" || conf.AppSecret == "" {
		return &wechatSession{OpenID: "dev_openid_" + stableCodePart(code)}, nil
	}
	endpoint := "https://api.weixin.qq.com/sns/jscode2session"
	q := url.Values{}
	q.Set("appid", conf.AppID)
	q.Set("secret", conf.AppSecret)
	q.Set("js_code", code)
	q.Set("grant_type", "authorization_code")
	var out wechatSession
	if err := getJSON(endpoint+"?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("wechat jscode2session failed: %s", out.ErrMsg)
	}
	if out.OpenID == "" {
		return nil, fmt.Errorf("wechat openid empty")
	}
	return &out, nil
}

func wxPhoneNumber(phoneCode string) (string, error) {
	conf := settings.Conf.MiniWechat
	if conf == nil || conf.AppID == "" || conf.AppSecret == "" {
		return "", fmt.Errorf("mini wechat appid/secret not configured")
	}
	token, err := wxAccessToken()
	if err != nil {
		return "", err
	}
	endpoint := "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=" + url.QueryEscape(token)
	reqBody := strings.NewReader(`{"code":"` + phoneCode + `"}`)
	resp, err := http.Post(endpoint, "application/json", reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out wechatPhone
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("wechat phone failed: %s", out.ErrMsg)
	}
	if out.PhoneInfo.PurePhoneNumber != "" {
		return out.PhoneInfo.PurePhoneNumber, nil
	}
	return out.PhoneInfo.PhoneNumber, nil
}

func wxAccessToken() (string, error) {
	conf := settings.Conf.MiniWechat
	endpoint := "https://api.weixin.qq.com/cgi-bin/token"
	q := url.Values{}
	q.Set("grant_type", "client_credential")
	q.Set("appid", conf.AppID)
	q.Set("secret", conf.AppSecret)
	var out struct {
		AccessToken string `json:"access_token"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := getJSON(endpoint+"?"+q.Encode(), &out); err != nil {
		return "", err
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("wechat access_token failed: %s", out.ErrMsg)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("wechat access_token empty")
	}
	return out.AccessToken, nil
}

func douyinCodeToSession(code string) (*douyinSession, error) {
	conf := settings.Conf.MiniDouyin
	if conf == nil || conf.AppID == "" || conf.AppSecret == "" {
		return nil, fmt.Errorf("mini douyin appid/secret not configured")
	}
	body, _ := json.Marshal(map[string]string{"appid": conf.AppID, "secret": conf.AppSecret, "code": code})
	resp, err := http.Post("https://developer.toutiao.com/api/apps/v2/jscode2session", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("douyin code2session http status %d", resp.StatusCode)
	}
	var out douyinSession
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ErrNo != 0 || out.Data.OpenID == "" || out.Data.SessionKey == "" {
		return nil, fmt.Errorf("douyin code2session failed: %s", out.ErrTips)
	}
	return &out, nil
}

// alipayCodeToSession 用 authCode 换取支付宝稳定 user_id。支付宝网关要求请求
// 参数以应用私钥做 RSA2 签名，私钥始终只存在于服务端。
func alipayCodeToSession(authCode string) (*alipayOAuthSession, error) {
	conf := settings.Conf.MiniAlipay
	if conf == nil || conf.AppID == "" || strings.TrimSpace(conf.AppPrivateKey) == "" {
		return nil, fmt.Errorf("mini alipay appid/private key not configured")
	}
	params := url.Values{
		"app_id":     {conf.AppID},
		"method":     {"alipay.system.oauth.token"},
		"format":     {"JSON"},
		"charset":    {"utf-8"},
		"sign_type":  {"RSA2"},
		"timestamp":  {time.Now().Format("2006-01-02 15:04:05")},
		"version":    {"1.0"},
		"grant_type": {"authorization_code"},
		"code":       {authCode},
	}
	sign, err := signAlipayParams(params, conf.AppPrivateKey)
	if err != nil {
		return nil, err
	}
	params.Set("sign", sign)
	resp, err := http.PostForm("https://openapi.alipay.com/gateway.do", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("alipay oauth http status %d", resp.StatusCode)
	}
	var result struct {
		Response alipayOAuthSession `json:"alipay_system_oauth_token_response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Response.Code != "10000" || result.Response.UserID == "" {
		return nil, fmt.Errorf("alipay oauth failed: %s", result.Response.Msg)
	}
	return &result.Response, nil
}

func signAlipayParams(params url.Values, privateKeyPEM string) (string, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "sign" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params.Get(key))
	}
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("invalid alipay app private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return "", err
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("alipay app private key must be RSA")
		}
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "&")))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func createDouyinPendingToken(openID, sessionKey string) string {
	token := randomHex(32)
	douyinPendingTokens.Lock()
	douyinPendingTokens.store[token] = douyinPendingSession{OpenID: openID, SessionKey: sessionKey, ExpiresAt: time.Now().Add(10 * time.Minute)}
	douyinPendingTokens.Unlock()
	return token
}

func consumeDouyinPendingToken(token string) (douyinPendingSession, bool) {
	douyinPendingTokens.Lock()
	defer douyinPendingTokens.Unlock()
	session, ok := douyinPendingTokens.store[token]
	delete(douyinPendingTokens.store, token)
	return session, ok && time.Now().Before(session.ExpiresAt)
}

func decryptDouyinPhone(sessionKey, encryptedData, iv string) (string, error) {
	key, err := decodeBase64(sessionKey)
	if err != nil || len(key) != aes.BlockSize {
		return "", fmt.Errorf("invalid douyin session key")
	}
	cipherText, err := decodeBase64(encryptedData)
	if err != nil || len(cipherText) == 0 || len(cipherText)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid encrypted phone data")
	}
	initializationVector, err := decodeBase64(iv)
	if err != nil || len(initializationVector) != aes.BlockSize {
		return "", fmt.Errorf("invalid phone iv")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(block, initializationVector).CryptBlocks(plain, cipherText)
	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", err
	}
	var out douyinPhone
	if err := json.Unmarshal(plain, &out); err != nil {
		return "", err
	}
	if settings.Conf.MiniDouyin.AppID != out.Watermark.AppID {
		return "", fmt.Errorf("douyin phone appid mismatch")
	}
	if out.PurePhoneNumber != "" {
		return out.PurePhoneNumber, nil
	}
	return out.PhoneNumber, nil
}

func douyinPhoneNumberByCode(phoneCode string) (string, error) {
	if strings.TrimSpace(phoneCode) == "" {
		return "", fmt.Errorf("douyin phone code required")
	}
	token, err := douyinClientToken()
	if err != nil {
		return "", err
	}
	var response struct {
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := postDouyinJSON("https://open.douyin.com/api/apps/v1/get_phonenumber_info/", map[string]string{"code": phoneCode}, map[string]string{"access-token": token}, &response); err != nil {
		return "", err
	}
	cipherText, err := douyinEncryptedPhonePayload(response.Data)
	if err != nil {
		return "", err
	}
	plain, err := decryptDouyinRSAPayload(cipherText)
	if err != nil {
		return "", err
	}
	var phone douyinPhone
	if err := json.Unmarshal(plain, &phone); err != nil {
		return "", err
	}
	if phone.Watermark.AppID != settings.Conf.MiniDouyin.AppID {
		return "", fmt.Errorf("douyin phone appid mismatch")
	}
	if phone.PurePhoneNumber != "" {
		return phone.PurePhoneNumber, nil
	}
	return phone.PhoneNumber, nil
}

func douyinClientToken() (string, error) {
	douyinClientTokens.Lock()
	defer douyinClientTokens.Unlock()
	if douyinClientTokens.value.Token != "" && time.Now().Add(5*time.Minute).Before(douyinClientTokens.value.ExpiresAt) {
		return douyinClientTokens.value.Token, nil
	}
	conf := settings.Conf.MiniDouyin
	if conf == nil || conf.AppID == "" || conf.AppSecret == "" {
		return "", fmt.Errorf("mini douyin appid/secret not configured")
	}
	var response struct {
		Data struct {
			AccessToken string `json:"access_token"`
			ErrorCode   int    `json:"error_code"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := postDouyinJSON("https://open.douyin.com/oauth/client_token/", map[string]string{"grant_type": "client_credential", "client_key": conf.AppID, "client_secret": conf.AppSecret}, nil, &response); err != nil {
		return "", err
	}
	if response.Data.ErrorCode != 0 || response.Data.AccessToken == "" {
		return "", fmt.Errorf("douyin client token failed")
	}
	expiresIn := response.Data.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	douyinClientTokens.value = douyinClientTokenCache{Token: response.Data.AccessToken, ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second)}
	return response.Data.AccessToken, nil
}

func postDouyinJSON(endpoint string, body any, headers map[string]string, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("douyin http status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func douyinEncryptedPhonePayload(raw json.RawMessage) (string, error) {
	var value string
	if json.Unmarshal(raw, &value) == nil && value != "" {
		return value, nil
	}
	var object struct {
		EncryptedData  string `json:"encrypted_data"`
		EncryptedData2 string `json:"encryptedData"`
	}
	if json.Unmarshal(raw, &object) == nil {
		if object.EncryptedData != "" {
			return object.EncryptedData, nil
		}
		if object.EncryptedData2 != "" {
			return object.EncryptedData2, nil
		}
	}
	return "", fmt.Errorf("douyin phone response missing encrypted data")
}

func decryptDouyinRSAPayload(cipherText string) ([]byte, error) {
	conf := settings.Conf.MiniDouyin
	if conf == nil || strings.TrimSpace(conf.PhonePrivateKeyPEM) == "" {
		return nil, fmt.Errorf("douyin phone private key not configured")
	}
	block, _ := pem.Decode([]byte(conf.PhonePrivateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid douyin phone private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return nil, err
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("douyin phone private key must be RSA")
		}
	}
	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, key, data)
}

func decodeBase64(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, fmt.Errorf("invalid encrypted phone padding")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, fmt.Errorf("invalid encrypted phone padding")
	}
	for _, item := range value[len(value)-padding:] {
		if int(item) != padding {
			return nil, fmt.Errorf("invalid encrypted phone padding")
		}
	}
	return value[:len(value)-padding], nil
}

func getJSON(endpoint string, out any) error {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func stableCodePart(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return randomHex(8)
	}
	if len(code) > 24 {
		return code[:24]
	}
	return code
}
