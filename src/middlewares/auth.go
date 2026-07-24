package middlewares

import (
	"net/http"
	"yonghemolimis/src/pkgs/session"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "mis_sid"

// AuthRequired 登录鉴权中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, err := c.Cookie(SessionCookieName)
		if err != nil || sid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未登录或会话已过期",
			})
			c.Abort()
			return
		}

		sess := session.Get(sid)
		if sess == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "会话已过期，请重新登录",
			})
			c.Abort()
			return
		}

		c.Set("userID", sess.AdminID)
		c.Set("username", sess.Username)
		c.Set("roles", sess.Roles)
		c.Set("permissions", sess.Permissions)
		c.Next()
	}
}
