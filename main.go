package main

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// 用户会话信息结构
type UserSession struct {
	IsLogin    bool                   `json:"is_login"`
	OpenID     string                 `json:"openid"`
	UserInfo   map[string]interface{} `json:"user_info"`
	LastActive time.Time              `json:"last_active"`
}

var (
	MaxSessionDuration = 10 * time.Minute
)

func main() {
	r := gin.Default()

	// 加载模板
	r.LoadHTMLGlob("templates/*")

	// 配置Session存储

	gob.Register(UserSession{})
	store := cookie.NewStore([]byte("secret-key-123456")) // 生产环境使用更复杂的密钥
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   int(MaxSessionDuration.Seconds()), //86400 * 7, // 7天
		HttpOnly: true,
		Secure:   false, // 本地开发设为false，生产环境设为true
	})
	//store.RegisterType(UserSession{})
	r.Use(sessions.Sessions("qq_session", store))

	// 路由定义
	r.GET("/", loginHandler)
	r.GET("/login", loginHandler)
	r.GET("/dashboard", SessionAuthMiddleware(), dashboardHandler)
	r.GET("/d2", SessionAuthMiddleware(), dashboardHandler)
	r.GET("/logout", logoutHandler)

	r.GET("/getdata", getdataHandler)

	NewQQLoginHandler(r)
	// 启动服务器
	r.Run(":9090")
}

func getdataHandler(c *gin.Context) {
	// 	[
	//     {"id":1, "name":"bob", "age":"23"},
	//     {"id":2, "name":"jim", "age":"45"},
	//     {"id":3, "name":"steve", "age":"32"}
	// ]
	//c.Writer.Write([]byte(`[{"id":1, "name":"bob", "age":"23"},{"id":2, "name":"jim", "age":"45"},{"id":3, "name":"steve", "age":"32"}]`))
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	users := []User{
		{ID: 1, Name: "bob", Age: 23},
		{ID: 2, Name: "jim", Age: 45},
		{ID: 3, Name: "steve", Age: 32},
	}

	c.JSON(http.StatusOK, users)
	//c.JSON(http.StatusOK, `[{"id":1, "name":"bob", "age":"23"},{"id":2, "name":"jim", "age":"45"},{"id":3, "name":"steve", "age":"32"}]`)
	// c.JSON(http.StatusOK, gin.H{
	// 	"id": 1, "name": "bob", "age": "23",
	// })
}

// 登录页
func loginHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "tabulator_test.html", nil)
}

// 仪表盘页面
func dashboardHandler(c *gin.Context) {
	session := sessions.Default(c)
	userSession := session.Get("user").(UserSession)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"user":       userSession.UserInfo,
		"lastActive": userSession.LastActive,
		"expired_in": userSession.LastActive.Add(MaxSessionDuration),
		"now_time":   time.Now(),
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
func CheckSessionValid(c *gin.Context) bool {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		return false
	}
	userSession, ok := user.(UserSession)
	if !ok || !userSession.IsLogin {
		return false
	}
	if time.Since(userSession.LastActive) > MaxSessionDuration {
		return false
	}
	return true
}
func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CheckSessionValid(c) {
			//get  c.path
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			redirectURL := fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.Request.RequestURI)
			fmt.Println("SessionAuthMiddleware redirectURL:", redirectURL)
			c.Redirect(http.StatusFound, "/toLogin?redirect_url="+redirectURL)
			c.Abort()
			return
		}
		c.Next()
	}
}
