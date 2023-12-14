package gm

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type loginArgs struct {
	Username  string
	Password  string
	AutoLogin bool
	LoginType string
}

func Login(c *gin.Context, data any) (any, error) {
	login := data.(*loginArgs)
	info := &dao.GmAccount{Username: "empiregame"}
	if login.Password != internal.Config().ProductKey {
		dbInfo, err := dao.GetGmAccount(login.Username)
		if err != nil {
			return nil, err
		}
		info = dbInfo

		sum := md5.Sum([]byte(login.Password))
		password := hex.EncodeToString(sum[:])
		if info.Password == "" || info.Password != password {
			return M{
				"status": "error",
				"type":   "account",
			}, nil
		}
	}

	token := generateToken(c, &dao.GmAccount{Id: info.Id, Username: info.Username})
	return M{
		"status": "ok",
		"name":   info.Username,
		"type":   "account",
		"userid": info.Id,
		"token":  token,
		"title":  info.Username,
	}, nil
}

// token生成器
// md 为上面定义好的middleware中间件
func generateToken(c *gin.Context, info *dao.GmAccount) string {
	// 构造SignKey: 签名和解签名需要使用一个值
	j := defaultJWT
	// 构造用户claims信息(负荷)
	claims := CustomClaims{
		Id:       info.Id,
		Username: info.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour)),          // 签名生效时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 签名过期时间
			Issuer:    "empiregame.cn",                                         // 签名颁发者
		},
	}
	// 根据claims生成token对象
	token, err := j.CreateToken(claims)
	if err != nil {
		c.JSON(200, gin.H{
			"status": -1,
			"msg":    err.Error(),
			"data":   nil,
		})
		return ""
	}
	return token
}

func queryCurrentUser(c *Context) {
	value, ok := c.Get("claims")
	if !ok {
		c.JSON(200, M{
			"Code":      1,
			"Msg":       "登陆会话已过期",
			"IsExpired": true,
		})
		return
	}

	claims := value.(*CustomClaims)
	account, err := dao.GetGmAccount(claims.Username)
	if err != nil {
		c.JSON(200, M{
			"Code": 1,
			"Msg":  err.Error(),
		})
		return
	}
	c.JSON(200, M{
		"Code": 0,
		"Msg":  "ok",
		"data": M{
			// "avatar":  "https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png",
			"name":      claims.Username,
			"type":      "account",
			"userid":    claims.Id,
			"title":     claims.Username,
			"access":    "admin",
			"country":   "CN",
			"localTime": time.Now().Format(internal.LongDateFmt),
			"menus":     account.Menus,
		},
	})
}

type userReq struct {
	Action string
	dao.GmAccount
}

func handleUser(c *Context, data any) (any, error) {
	req := data.(*userReq)

	var err error
	switch req.Action {
	case "add":
		_, err = dao.AddGmAccount(&req.GmAccount)
	case "update":
		err = dao.UpdateGmAccount(&req.GmAccount)
	case "delete":
		err = dao.DeleteGmAccount(req.Id)
	case "query":
		account, err := dao.GetGmAccount(req.Username)
		return account, err
	default:
		accounts, err := dao.QueryGmAccount()
		if err != nil {
			return nil, err
		}
		return M{"data": accounts}, nil

	}
	return M{}, err
}
