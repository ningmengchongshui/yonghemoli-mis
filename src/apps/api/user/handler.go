package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"yonghemolimis/src/apps/api/response"
	"yonghemolimis/src/middlewares"
	"yonghemolimis/src/pkgs/session"
	"yonghemolimis/src/settings"
)

const stateCookie = "yh_oidc_state"
const verifierCookie = "yh_oidc_verifier"

func OIDCAuthorize(c *gin.Context) {
	if !configured() {
		response.Error(c, 500, "OIDC 客户端未配置")
		return
	}
	var b struct {
		RedirectURL string `json:"redirect_uri" form:"redirect_uri" binding:"required"`
	}
	if c.ShouldBind(&b) != nil || b.RedirectURL != settings.Conf.OIDC.RedirectURL {
		response.Error(c, 400, "回调地址无效")
		return
	}
	state, e := random()
	if e != nil {
		response.Error(c, 500, "无法初始化登录")
		return
	}
	v, e := random()
	if e != nil {
		response.Error(c, 500, "无法初始化登录")
		return
	}
	sum := sha256.Sum256([]byte(v))
	secure := prod()
	c.SetCookie(stateCookie, state, 600, "/", "", secure, true)
	c.SetCookie(verifierCookie, v, 600, "/", "", secure, true)
	q := url.Values{"response_type": {"code"}, "client_id": {settings.Conf.OIDC.ClientID}, "redirect_uri": {settings.Conf.OIDC.RedirectURL}, "scope": {"openid profile email"}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}}
	response.OK(c, gin.H{"authorizeURL": strings.TrimRight(settings.Conf.OIDC.IssuerURL, "/") + "/oauth/authorize?" + q.Encode()})
}
func OIDCCallback(c *gin.Context) {
	var b struct {
		Code  string `json:"code" form:"code" binding:"required"`
		State string `json:"state" form:"state" binding:"required"`
	}
	if c.ShouldBind(&b) != nil {
		response.Error(c, 400, "授权响应无效")
		return
	}
	s, e := c.Cookie(stateCookie)
	v, e2 := c.Cookie(verifierCookie)
	if e != nil || e2 != nil || s == "" || s != b.State || v == "" {
		response.Error(c, 403, "登录请求已失效，请重新发起登录")
		return
	}
	c.SetCookie(stateCookie, "", -1, "/", "", prod(), true)
	c.SetCookie(verifierCookie, "", -1, "/", "", prod(), true)
	u, e := exchange(b.Code, v)
	if e != nil {
		response.Error(c, 401, "OIDC 认证失败："+e.Error())
		return
	}
	sid := session.Create(u.ID, u.Name, u.Email, "", u.Roles, u.Permissions)
	c.SetCookie(middlewares.SessionCookieName, sid, 86400*7, "/", "", prod(), true)
	response.OK(c, gin.H{"user": gin.H{"id": u.ID, "username": u.Name, "email": u.Email, "roles": u.Roles, "permissions": u.Permissions}})
}
func OIDCSession(c *gin.Context) {
	sid, _ := c.Cookie(middlewares.SessionCookieName)
	x := session.Get(sid)
	if sid == "" || x == nil {
		response.FailCode(c, 401, "未登录")
		return
	}
	response.OK(c, gin.H{"user": gin.H{"id": x.AdminID, "username": x.Username, "email": x.Email, "roles": x.Roles, "permissions": x.Permissions}})
}
func Logout(c *gin.Context) {
	sid, _ := c.Cookie(middlewares.SessionCookieName)
	session.Destroy(sid)
	c.SetCookie(middlewares.SessionCookieName, "", -1, "/", "", prod(), true)
	response.OKMsg(c, "已登出")
}
func Me(c *gin.Context) {
	response.OK(c, gin.H{"user": gin.H{"id": c.GetUint("userID"), "username": c.GetString("username"), "roles": c.GetStringSlice("roles"), "permissions": c.GetStringSlice("permissions")}})
}

type oidcUser struct {
	ID                 uint
	Name, Email        string
	Roles, Permissions []string
}

func exchange(code, v string) (*oidcUser, error) {
	issuer := strings.TrimRight(settings.Conf.OIDC.IssuerURL, "/")
	f := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {settings.Conf.OIDC.ClientID}, "client_secret": {settings.Conf.OIDC.ClientSecret}, "redirect_uri": {settings.Conf.OIDC.RedirectURL}, "code_verifier": {v}}
	cl := &http.Client{Timeout: 10 * time.Second}
	r, e := cl.PostForm(issuer+"/oauth/token", f)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	var t struct {
		AccessToken string `json:"access_token"`
	}
	if r.StatusCode != 200 || json.NewDecoder(r.Body).Decode(&t) != nil || t.AccessToken == "" {
		return nil, fmt.Errorf("OIDC token 无效")
	}
	req, _ := http.NewRequest("GET", issuer+"/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	p, e := cl.Do(req)
	if e != nil {
		return nil, e
	}
	defer p.Body.Close()
	var x struct {
		Subject           string   `json:"sub"`
		Name              string   `json:"name"`
		PreferredUsername string   `json:"preferred_username"`
		Email             string   `json:"email"`
		Roles             []string `json:"roles"`
		Permissions       []string `json:"permissions"`
	}
	if p.StatusCode != 200 || json.NewDecoder(p.Body).Decode(&x) != nil {
		return nil, fmt.Errorf("OIDC userinfo 无效")
	}
	id, e := strconv.ParseUint(x.Subject, 10, 64)
	if e != nil || id == 0 {
		return nil, fmt.Errorf("OIDC subject 无效")
	}
	if x.PreferredUsername != "" {
		x.Name = x.PreferredUsername
	}
	return &oidcUser{uint(id), x.Name, x.Email, x.Roles, x.Permissions}, nil
}
func configured() bool {
	return settings.Conf.OIDC != nil && settings.Conf.OIDC.IssuerURL != "" && settings.Conf.OIDC.ClientID != "" && settings.Conf.OIDC.ClientSecret != "" && settings.Conf.OIDC.RedirectURL != ""
}
func random() (string, error) {
	b := make([]byte, 32)
	_, e := rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b), e
}
func prod() bool { return settings.Conf.Mode == "release" || settings.Conf.Mode == "production" }
