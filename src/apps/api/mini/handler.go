package miniapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"yonghemolimis/src/apps/api/chatws"
	"yonghemolimis/src/dao/db"

	"github.com/gin-gonic/gin"
)

type R struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type User struct {
	ID          string `json:"id"`
	NickName    string `json:"nickName"`
	AvatarURL   string `json:"avatarUrl"`
	Signature   string `json:"signature"`
	Phone       string `json:"phone"`
	LastLoginAt string `json:"lastLoginAt"`
}

func RegisterRoutes(r gin.IRouter) {
	r.POST("/auth/wechat-login", WechatLogin)
	r.POST("/auth/douyin-login", DouyinLogin)
	r.POST("/auth/douyin-phone-login", DouyinPhoneLogin)
	r.POST("/auth/alipay-login", AlipayLogin)
	r.POST("/auth/phone-code", PhoneCode)
	r.POST("/auth/phone-login", PhoneLogin)
	r.GET("/users/me", BusinessUserMe)
	r.GET("/app-config", BusinessAppConfig)
	r.GET("/agreements/:type", BusinessAgreement)
	r.GET("/about", BusinessAbout)
	r.GET("/faqs", ListBusinessFAQs)
	r.GET("/services", ListBusinessServices)
	r.GET("/caregivers", ListBusinessCaregivers)
	r.GET("/caregivers/:id", GetBusinessCaregiver)
	r.POST("/demands", CreateBusinessDemand)
	r.POST("/resumes", CreateBusinessResume)
	r.GET("/chat/session", ChatSession)
	r.GET("/chat/messages", ChatMessages)
	r.POST("/chat/messages", CreateChatMessage)
}

func requestUser(c *gin.Context) (User, string) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	userID := miniUserIDFromToken(token)
	if userID == "" {
		return User{}, ""
	}
	row, err := db.GetMiniUserProfile(userID)
	if err != nil {
		return User{}, ""
	}
	return userFromDO(*row), row.ID
}

func requestUserID(c *gin.Context) string { _, userID := requestUser(c); return userID }

func WechatLogin(c *gin.Context) {
	var req struct {
		Code      string `json:"code"`
		PhoneCode string `json:"phoneCode"`
		NickName  string `json:"nickName"`
		AvatarURL string `json:"avatarUrl"`
	}
	if c.ShouldBindJSON(&req) != nil {
		businessFail(c, 400, 40000, "请求参数错误")
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		businessFail(c, 400, 40000, "微信登录凭证不能为空")
		return
	}
	wxSession, err := wxCodeToSession(req.Code)
	if err != nil {
		businessFail(c, 400, 40002, "微信登录凭证错误或已失效")
		return
	}
	if row, err := db.GetMiniUserByOpenID(wxSession.OpenID); err == nil && strings.TrimSpace(row.Phone) != "" {
		if row.Status != db.CustomerStatusActive {
			businessFail(c, 403, 40300, "账号已被禁用")
			return
		}
		row.LastLoginAt = time.Now().Format("2006-01-02 15:04:05")
		row.UpdatedAt = time.Now()
		if db.UpsertMiniUserProfile(row) != nil {
			businessFail(c, 500, 50000, "登录失败")
			return
		}
		loginSuccess(c, row)
		return
	}
	if strings.TrimSpace(req.PhoneCode) == "" {
		businessFail(c, 400, 40000, "phoneCode 不能为空")
		return
	}
	phone, err := wxPhoneNumber(req.PhoneCode)
	if err != nil {
		businessFail(c, 400, 40002, "微信手机号授权错误或已失效")
		return
	}
	if !mainlandPhonePattern.MatchString(phone) {
		businessFail(c, 400, 40001, "手机号格式不正确")
		return
	}
	row, err := ensureMiniUser(wxSession.OpenID, req.NickName, req.AvatarURL, phone)
	if err != nil {
		businessFail(c, 409, 40900, "该手机号或微信账号已绑定其他用户")
		return
	}
	loginSuccess(c, row)
}

// DouyinLogin 使用 tt.login 返回的 code 建立抖音身份。已绑定手机号的用户
// 直接登录；首次用户则返回短时 authToken，供手机号组件回调继续完成绑定。
func DouyinLogin(c *gin.Context) {
	var req struct {
		Code      string `json:"code"`
		NickName  string `json:"nickName"`
		AvatarURL string `json:"avatarUrl"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Code) == "" {
		businessFail(c, 400, 40000, "抖音登录凭证不能为空")
		return
	}
	session, err := douyinCodeToSession(req.Code)
	if err != nil {
		businessFail(c, 400, 40002, "抖音登录凭证错误、已失效或服务未配置")
		return
	}
	if row, err := db.GetMiniUserByDouyinOpenID(session.Data.OpenID); err == nil {
		if row.Status != db.CustomerStatusActive {
			businessFail(c, 403, 40300, "账号已被禁用")
			return
		}
		row.LastLoginAt = time.Now().Format("2006-01-02 15:04:05")
		row.UpdatedAt = time.Now()
		if err := db.UpsertMiniUserProfile(row); err != nil {
			businessFail(c, 500, 50000, "登录失败")
			return
		}
		loginSuccess(c, row)
		return
	}
	businessOK(c, http.StatusOK, gin.H{
		"needPhoneAuth": true,
		"authToken":     createDouyinPendingToken(session.Data.OpenID, session.Data.SessionKey),
	})
}

// DouyinPhoneLogin 解密 getPhoneNumber 回调中的手机号并绑定抖音 openid。
// 不能在这里重新 tt.login，否则 session_key 会变化，导致解密失败。
func DouyinPhoneLogin(c *gin.Context) {
	var req struct {
		AuthToken     string `json:"authToken"`
		PhoneCode     string `json:"phoneCode"`
		EncryptedData string `json:"encryptedData"`
		IV            string `json:"iv"`
		NickName      string `json:"nickName"`
		AvatarURL     string `json:"avatarUrl"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.AuthToken) == "" || (strings.TrimSpace(req.PhoneCode) == "" && (strings.TrimSpace(req.EncryptedData) == "" || strings.TrimSpace(req.IV) == "")) {
		businessFail(c, 400, 40000, "手机号授权参数不完整")
		return
	}
	pending, ok := consumeDouyinPendingToken(req.AuthToken)
	if !ok {
		businessFail(c, 400, 40002, "手机号授权已失效，请重新进行抖音登录")
		return
	}
	phone, err := douyinPhoneNumberByCode(req.PhoneCode)
	if strings.TrimSpace(req.PhoneCode) == "" {
		phone, err = decryptDouyinPhone(pending.SessionKey, req.EncryptedData, req.IV)
	}
	if err != nil || !mainlandPhonePattern.MatchString(phone) {
		businessFail(c, 400, 40002, "抖音手机号授权失败")
		return
	}
	row, err := ensureMiniUserByDouyin(pending.OpenID, req.NickName, req.AvatarURL, phone)
	if err != nil {
		businessFail(c, 409, 40900, "该手机号或抖音账号已绑定其他用户")
		return
	}
	loginSuccess(c, row)
}

// AlipayLogin 使用 my.getAuthCode 的 authCode 换取支付宝 user_id。首次用户
// 可直接建立仅第三方身份的账户，手机号后续按用户主动授权再绑定。
func AlipayLogin(c *gin.Context) {
	var req struct {
		AuthCode  string `json:"authCode"`
		NickName  string `json:"nickName"`
		AvatarURL string `json:"avatarUrl"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.AuthCode) == "" {
		businessFail(c, 400, 40000, "支付宝授权凭证不能为空")
		return
	}
	session, err := alipayCodeToSession(req.AuthCode)
	if err != nil {
		businessFail(c, 400, 40002, "支付宝授权凭证错误、已失效或服务未配置")
		return
	}
	row, err := db.GetMiniUserByAlipayUserID(session.UserID)
	created := false
	if err != nil {
		row = &db.CustomerDO{ID: newMiniUserID(), AlipayUserID: session.UserID, Nickname: "支付宝用户", Status: db.CustomerStatusActive, CreatedAt: time.Now()}
		if strings.TrimSpace(req.NickName) != "" {
			row.Nickname = strings.TrimSpace(req.NickName)
		}
		if strings.TrimSpace(req.AvatarURL) != "" {
			row.Avatar = strings.TrimSpace(req.AvatarURL)
		}
		row.LastLoginAt = time.Now().Format("2006-01-02 15:04:05")
		row.UpdatedAt = time.Now()
		if err := db.CreateMiniUserWithoutPhone(row); err != nil {
			businessFail(c, 500, 50000, "创建支付宝用户失败")
			return
		}
		created = true
	}
	if row.Status != db.CustomerStatusActive {
		businessFail(c, 403, 40300, "账号已被禁用")
		return
	}
	row.LastLoginAt = time.Now().Format("2006-01-02 15:04:05")
	row.UpdatedAt = time.Now()
	if !created {
		if err := db.UpsertMiniUserProfile(row); err != nil {
			businessFail(c, 500, 50000, "登录失败")
			return
		}
	}
	loginSuccess(c, row)
}

func loginSuccess(c *gin.Context, row *db.CustomerDO) {
	user := userFromDO(*row)
	user.Phone = maskPhone(row.Phone)
	businessOK(c, http.StatusOK, gin.H{"token": createMiniToken(row.ID), "expiresIn": 30 * 24 * 60 * 60, "user": user, "isBoundPhone": strings.TrimSpace(row.Phone) != ""})
}

func PhoneCode(c *gin.Context)  { phoneCode(c) }
func PhoneLogin(c *gin.Context) { phoneLogin(c) }

func ChatSession(c *gin.Context) {
	user, userID := requestUser(c)
	if userID == "" {
		businessFail(c, 401, 40100, "未登录或 Token 缺失")
		return
	}
	sessionID := c.DefaultQuery("sessionId", "chat_"+userID)
	row, err := chatws.TouchSession(sessionID, userID, user.NickName)
	if err != nil {
		businessFail(c, 500, 50000, "创建客服会话失败")
		return
	}
	businessOK(c, 200, gin.H{"item": row})
}

func ChatMessages(c *gin.Context) {
	_, userID := requestUser(c)
	if userID == "" {
		businessFail(c, 401, 40100, "未登录或 Token 缺失")
		return
	}
	sessionID := c.DefaultQuery("sessionId", "chat_"+userID)
	rows, err := db.ListChatMessages(sessionID)
	if err != nil {
		businessFail(c, 500, 50000, "查询客服消息失败")
		return
	}
	businessOK(c, 200, gin.H{"list": rows})
}

func CreateChatMessage(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
		Content   string `json:"content" binding:"required"`
		MsgType   string `json:"msgType"`
	}
	if c.ShouldBindJSON(&req) != nil {
		businessFail(c, 400, 40000, "消息内容不能为空")
		return
	}
	user, userID := requestUser(c)
	if userID == "" {
		businessFail(c, 401, 40100, "未登录或 Token 缺失")
		return
	}
	if req.SessionID == "" {
		req.SessionID = "chat_" + userID
	}
	row, err := chatws.CreateMiniMessage(req.SessionID, userID, user.NickName, req.MsgType, req.Content)
	if err != nil {
		businessFail(c, 500, 50000, "发送客服消息失败")
		return
	}
	businessOK(c, 200, gin.H{"item": row})
}

func ensureMiniUser(openID, nickName, avatarURL, phone string) (*db.CustomerDO, error) {
	if strings.TrimSpace(phone) == "" {
		return nil, fmt.Errorf("phone required")
	}
	row, err := db.GetMiniUserByPhone(phone)
	if err != nil {
		row = &db.CustomerDO{ID: newMiniUserID(), Phone: phone, Nickname: maskPhone(phone), Status: db.CustomerStatusActive, CreatedAt: time.Now()}
	}
	if row.OpenID != "" && openID != "" && row.OpenID != openID {
		return nil, fmt.Errorf("phone already bound")
	}
	if row.OpenID == "" {
		row.OpenID = openID
	}
	if strings.TrimSpace(nickName) != "" {
		row.Nickname = strings.TrimSpace(nickName)
	}
	if avatarURL != "" {
		row.Avatar = avatarURL
	}
	row.Phone = phone
	row.LastLoginAt = time.Now().Format("2006-01-02 15:04:05")
	row.UpdatedAt = time.Now()
	return row, db.UpsertMiniUserProfile(row)
}

func ensureMiniUserByDouyin(openID, nickName, avatarURL, phone string) (*db.CustomerDO, error) {
	if strings.TrimSpace(phone) == "" || strings.TrimSpace(openID) == "" {
		return nil, fmt.Errorf("douyin openid and phone required")
	}
	if row, err := db.GetMiniUserByDouyinOpenID(openID); err == nil {
		if row.Phone != phone {
			return nil, fmt.Errorf("douyin account already bound")
		}
		return row, nil
	}
	row, err := db.GetMiniUserByPhone(phone)
	if err != nil {
		row = &db.CustomerDO{ID: newMiniUserID(), Phone: phone, Nickname: maskPhone(phone), Status: db.CustomerStatusActive, CreatedAt: time.Now()}
	}
	if row.DouyinOpenID != "" && row.DouyinOpenID != openID {
		return nil, fmt.Errorf("phone already bound")
	}
	row.DouyinOpenID = openID
	if strings.TrimSpace(nickName) != "" {
		row.Nickname = strings.TrimSpace(nickName)
	}
	if strings.TrimSpace(avatarURL) != "" {
		row.Avatar = strings.TrimSpace(avatarURL)
	}
	row.LastLoginAt = time.Now().Format("2006-01-02 15:04:05")
	row.UpdatedAt = time.Now()
	return row, db.UpsertMiniUserProfile(row)
}

func newMiniUserID() string {
	return "U" + time.Now().Format("20060102150405") + fmt.Sprintf("%03d", time.Now().Nanosecond()%1000)
}
func userFromDO(row db.CustomerDO) User {
	return User{ID: row.ID, NickName: row.Nickname, AvatarURL: row.Avatar, Signature: row.Signature, Phone: row.Phone, LastLoginAt: row.LastLoginAt}
}
