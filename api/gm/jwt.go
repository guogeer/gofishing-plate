package gm

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gofishing-plate/internal"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/guogeer/quasar/log"
)

var defaultJWT *JWT

// 定义一个jwt对
type JWT struct {
	// 声明签名信息
	SigningKey []byte
}

func init() {
	defaultJWT = &JWT{SigningKey: []byte(internal.Config().ProductKey)}
}

// 自定义有效载荷(这里采用自定义的用户名和id作为有效载荷的一部分)
type CustomClaims struct {
	Id       int    `json:"id"`
	Username string `json:"userName"`
	// RegisteredClaims结构体实现了Claims接口(Valid()函数)
	jwt.RegisteredClaims
}

// 调用jwt-go库生成token
// 指定编码的算法为jwt.SigningMethodHS256
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	// 返回一个token的结构体指针
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// token解码
func (j *JWT) ParserToken(tokenString string) (*CustomClaims, error) {
	// 输入用户自定义的Claims结构体对象,token,以及自定义函数来解析token字符串为jwt的Token结构体指针
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return j.SigningKey, nil
	})

	if err != nil {
		// jwt.ValidationError 是一个无效token的错误结构
		if ve, ok := err.(*jwt.ValidationError); ok {
			// ValidationErrorMalformed是一个uint常量，表示token不可用
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, fmt.Errorf("无效的登陆会话")
				// ValidationErrorExpired表示Token过期
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, fmt.Errorf("登陆会话已过期")
				// ValidationErrorNotValidYet表示无效token
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, fmt.Errorf("无效的登陆会话")
			} else {
				return nil, fmt.Errorf("登陆会话不可用")
			}
		}
	}
	// 将token中的claims信息解析出来并断言成用户自定义的有效载荷结构
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("登陆会话无效")
}

// 定义一个JWTAuth的中间件
func JWTAuth(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 通过http header中的token解析来认证
		token := c.Request.Header.Get("Token")

		if token == "undefined" {
			token = ""
		}

		var err error
		var claims *CustomClaims
		if token != "" {
			if claims, err = defaultJWT.ParserToken(token); err == nil {
				c.Set("claims", claims)
			}
		}

		if !strings.HasPrefix(c.Request.URL.Path, prefix) {
			return
		}
		if token == "" {
			err = errors.New("无效的登陆会话")
		}

		if err != nil {
			// token无效错误
			log.Debugf("get token: %v error: %v", token, err)
			c.Header("Auth", url.QueryEscape(err.Error()))
			c.Abort()
			return
		}
	}
}
