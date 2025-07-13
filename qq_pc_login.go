package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// this code  from
// https://github.com/pibigstar/go-demo/blob/master/sdk/qq/qq_pc_login.go
const (
	AppId       = "101827468"
	AppKey      = "0d2d856e48e0ebf6b98e0d0c879fe74d"
	redirectURI = "http://127.0.0.1:9090/qqLogin"
)

type PrivateInfo struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
}
type QQLoginHandler struct {
}

func (h *QQLoginHandler) setupRoutes(r *gin.Engine) {
	r.GET("/toLogin", h.qqAuthHandler)
	r.GET("/qqLogin", h.qqCallbackHandler)
}

var (
	instance *QQLoginHandler
	once     sync.Once
)

func NewQQLoginHandler(r *gin.Engine) *QQLoginHandler {
	once.Do(func() {
		instance = &QQLoginHandler{}
		instance.setupRoutes(r)
	})
	return instance
}

// 1. Get Authorization Code

// 发起QQ登录
func (h *QQLoginHandler) qqAuthHandler(c *gin.Context) {
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", AppId)

	state := fmt.Sprintf("%d", time.Now().Unix()) // 简单的state生成方式
	params.Add("state", state)
	str := fmt.Sprintf("%s&redirect_uri=%s", params.Encode(), redirectURI)
	loginURL := fmt.Sprintf("%s?%s", "https://graph.qq.com/oauth2.0/authorize", str)

	//http.Redirect(w, r, loginURL, http.StatusFound)
	c.Redirect(http.StatusFound, loginURL)
}

// 2. Get Access Token
// func GetToken(w http.ResponseWriter, r *http.Request)

func (h *QQLoginHandler) qqCallbackHandler(c *gin.Context) {

	state := c.Query("state")
	if state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "缺少state参数"})
		return
	}
	// 获取授权码
	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "缺少授权码"})
		return
	}

	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("client_id", AppId)
	params.Add("client_secret", AppKey)
	params.Add("code", code)
	str := fmt.Sprintf("%s&redirect_uri=%s", params.Encode(), redirectURI)
	loginURL := fmt.Sprintf("%s?%s", "https://graph.qq.com/oauth2.0/token", str)

	// 使用授权码获取access token
	response, err := http.Get(loginURL)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "获取access token失败"})
	}
	defer response.Body.Close()

	bs, _ := ioutil.ReadAll(response.Body)
	body := string(bs)
	resultMap := convertToMap(body)

	info := &PrivateInfo{}
	info.AccessToken = resultMap["access_token"]
	info.RefreshToken = resultMap["refresh_token"]
	info.ExpiresIn = resultMap["expires_in"]

	openid, err := getQQOpenID(info.AccessToken)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "获取openid失败"})
		return
	}
	// 获取用户信息
	userInfo, err := getQQUserInfo(info.AccessToken, openid, AppId)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "获取用户信息失败"})
		return
	}

	// 设置Session
	session := sessions.Default(c)
	userSession := UserSession{
		IsLogin:  true,
		OpenID:   openid,
		UserInfo: userInfo,
	}
	session.Clear()
	session.Set("user", userSession)

	// 重新生成Session ID防止固定攻击

	if err := session.Save(); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "保存会话失败"})
		return
	}

	// 登录成功后重定向到仪表盘
	c.Redirect(http.StatusFound, "/dashboard")
}

// 3.获取QQ openid
func getQQOpenID(accessToken string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", accessToken))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// QQ返回的数据格式为: callback( {"client_id":"YOUR_APPID","openid":"YOUR_OPENID"} );
	// 需要处理这个JSONP格式
	var result struct {
		OpenID string `json:"openid"`
	}
	if err := json.Unmarshal(body[9:len(body)-3], &result); err != nil {
		return "", err
	}

	return result.OpenID, nil
}

// 4. 获取QQ用户信息
func getQQUserInfo(accessToken, openid, appID string) (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf(
		"https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		accessToken, appID, openid))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	// 添加openid到用户信息
	userInfo["openid"] = openid

	return userInfo, nil
}

func convertToMap(str string) map[string]string {
	var resultMap = make(map[string]string)
	values := strings.Split(str, "&")
	for _, value := range values {
		vs := strings.Split(value, "=")
		resultMap[vs[0]] = vs[1]
	}
	return resultMap
}
