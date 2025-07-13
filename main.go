package main

import (
	"encoding/gob"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// 用户会话信息结构
type UserSession struct {
	IsLogin  bool                   `json:"is_login"`
	OpenID   string                 `json:"openid"`
	UserInfo map[string]interface{} `json:"user_info"`
}

func main() {
	r := gin.Default()

	// 加载模板
	r.LoadHTMLGlob("templates/*")

	// 配置Session存储

	gob.Register(UserSession{})
	store := cookie.NewStore([]byte("secret-key-123456")) // 生产环境使用更复杂的密钥
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   60, //86400 * 7, // 7天
		HttpOnly: true,
		Secure:   false, // 本地开发设为false，生产环境设为true
	})
	//store.RegisterType(UserSession{})
	r.Use(sessions.Sessions("qq_session", store))

	// 路由定义
	r.GET("/", loginHandler)
	r.GET("/login", loginHandler)
	r.GET("/dashboard", authRequired(), dashboardHandler)
	r.GET("/logout", logoutHandler)

	NewQQLoginHandler(r)
	// 启动服务器
	r.Run(":9090")
}

// 首页
func homeHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

// 登录页
func loginHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

// 仪表盘页面
func dashboardHandler(c *gin.Context) {
	session := sessions.Default(c)
	userSession := session.Get("user").(UserSession)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"user": userSession.UserInfo,
	})
}

// 注销
func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

// 认证中间件
func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")

		if user == nil {
			// 未登录，重定向到登录页
			c.Redirect(http.StatusFound, "/toLogin")
			c.Abort()
			return
		}

		// 检查会话是否有效
		userSession, ok := user.(UserSession)
		if !ok || !userSession.IsLogin {
			c.Redirect(http.StatusFound, "/toLogin")
			c.Abort()
			return
		}

		// 用户已登录，继续处理请求
		c.Next()
	}
}
