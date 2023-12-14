package gm

import (
	"gofishing-plate/dao"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

func GetClientVersionList(c *Context, data any) (any, error) {
	log.Debugf("gm获取客户端版本列表....!")
	clientVersions, _ := dao.QueryAllClientVersion()

	return M{"data": clientVersions}, nil
}

func ClientVersionAdd(c *Context, data any) (any, error) {
	clientVersion := data.(*dao.ClientVersion)
	dao.AddClientVersion(clientVersion)

	// 向大厅发送GM更新消息
	cmd.Forward("hall", "FUNC_UpdateClientVersion", M{})
	return M{}, nil
}

func ClientVersionDelete(c *Context, data any) (any, error) {
	clientVersion := data.(*dao.ClientVersion)
	//log.Debugf("delete %v,%v", clientVersion.Id, clientVersion.ChanId)
	dao.DeleteClientVersion(clientVersion.Id)

	// 向大厅发送GM更新消息
	cmd.Forward("hall", "FUNC_UpdateClientVersion", M{})
	return M{}, nil
}

func ClientVersionUpdate(c *Context, data any) (any, error) {
	clientVersion := data.(*dao.ClientVersion)
	dao.UpdateClientVersion(clientVersion)

	// 向大厅发送GM更新消息
	cmd.Forward("hall", "FUNC_UpdateClientVersion", M{})
	return M{}, nil
}

func GetRemoteIP(c *Context, data any) (any, error) {
	return M{"IP": c.ClientIP()}, nil
}

type clientBundleReq struct {
	Method string
	dao.ClientBundle
}

// 分包资源
func clientBundle(c *Context, data any) (any, error) {
	req := data.(*clientBundleReq)

	var err error
	var clientBundles []*dao.ClientBundle
	switch req.Method {
	case "add":
		_, err = dao.AddClientBundle(&req.ClientBundle)
	case "update":
		err = dao.UpdateClientBundle(&req.ClientBundle)
	case "delete":
		err = dao.DeleteClientBundle(req.Id)
	default:
		clientBundles, err = dao.QueryClientBundle()
	}
	if err != nil {
		return nil, err
	}
	return M{"Code": 0, "Msg": "ok", "data": clientBundles}, nil
}
