package gm

import (
	"gofishing-plate/api"
	"gofishing-plate/dao"
)

type M = api.M
type Context = api.Context

type emptyArgs struct{}

const apiAdminURL = "/api/admin"

func handleAuth(url string, h api.Handler, data any) {
	api.HandleAPI(apiAdminURL+url, h, data)
}

func registerHandlers(r api.IRoutes) {
	r.Use(JWTAuth(apiAdminURL))
	//获取对应的权限
	//auth.POST("/queryMenusAndPermission", gm.GetMenusByPermission)
	//gm对应的版本管理
	handleAuth("/clientVersion/list", GetClientVersionList, (*dao.ClientVersion)(nil))
	handleAuth("/clientVersion/add", ClientVersionAdd, (*dao.ClientVersion)(nil))
	handleAuth("/clientVersion/update", ClientVersionUpdate, (*dao.ClientVersion)(nil))
	handleAuth("/clientVersion/delete", ClientVersionDelete, (*dao.ClientVersion)(nil))

	handleAuth("/ip", GetRemoteIP, (*emptyArgs)(nil))
	handleAuth("/online", queryOnline, (*onlineReq)(nil))
	handleAuth("/itemLog", queryItemLog, (*itemLogReq)(nil))
	handleAuth("/payOrder", queryPayOrder, (*payOrderReq)(nil))
	handleAuth("/configTable", handleConfigTable, (*configTableReq)(nil))
	handleAuth("/regUser", handleRegUser, (*regUserReq)(nil))
	handleAuth("/addItems", addItems, (*addItemsReq)(nil))
	handleAuth("/chan", queryChan, (*chanReq)(nil))
	handleAuth("/clientBundle", clientBundle, (*clientBundleReq)(nil))
	handleAuth("/user", handleUser, (*userReq)(nil))
	handleAuth("/maintain", handleMaintain, (*maintainReq)(nil))
	handleAuth("/mail", handleMail, (*mailReq)(nil))

	// gm后台登录无需验证
	api.HandleAPI("/api/login/account", Login, (*loginArgs)(nil))
	r.GET("/api/currentUser", queryCurrentUser)
}

func init() {

	api.PreloadRoutes(registerHandlers)
}
