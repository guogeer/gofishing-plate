package api

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type ProjectConfig struct {
	KiwiPay struct {
		Key     string
		TestKey string
	}
	Tapdb struct {
		AppId string
	}
	Thinkingdata struct {
		ServerURL string
		AppId     string
		TestAppId string
	}
}

var projectConfig ProjectConfig

func init() {
	os.MkdirAll(internal.Config().ResourcePath, 0755)
	os.MkdirAll(internal.Config().ResourcePath+"/tables", 0755)

	config.LoadFile("configs/project.json", &projectConfig)
}

func createFileName() string {
	now := time.Now()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
	)
}

// 向游戏内的玩家发送消息
func SendMsg(uid int, msgId string, body any) error {
	user, err := dao.GetRegUserInfo(uid)
	log.Debugf("sendMsg player %d msgId %s ServerName %s error %v", uid, msgId, user.ServerId, err)
	if err != nil {
		return err
	}

	if user.ServerId == "" {
		user.ServerId = "hall"
	}

	cmd.Route(user.ServerId, msgId, body)
	return nil
}

type plateArgs struct {
	IconName string
	Path     string

	InstallReferer string
}

// 保存FB头像
func savePlateIcon(c *Context, in any) (any, error) {
	args := in.(*plateArgs)
	u, err := url.Parse(args.Path)
	if err != nil {
		return nil, err
	}
	elems := strings.Split(u.Path, "/")
	if len(elems) == 0 {
		return nil, errors.New("invalid path")
	}
	os.MkdirAll(internal.Config().ResourcePath+"/"+u.Host, 0755)
	iconPath := internal.Config().ResourcePath + "/" + u.Host + "/" + args.IconName
	// 文件存在
	if _, err := os.Stat(iconPath); err == nil {
		return cmd.M{"Path": internal.Config().ResourceURL + "/" + iconPath}, nil
	}

	return cmd.M{"Path": internal.Config().ResourceURL + "/" + iconPath}, nil
}
