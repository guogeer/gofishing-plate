package api

import (
	"gofishing-plate/dao"

	"github.com/guogeer/quasar/cmd"
)

type onlineArgs struct {
	Servers []struct {
		ServerName string `json:"serverName,omitempty"`
		Id         int    `json:"id,omitempty"`
		Num        int    `json:"num,omitempty"`
	} `json:"servers,omitempty"`
}

func init() {
	cmd.BindFunc(ReportOnline, (*onlineArgs)(nil))
}

func ReportOnline(ctx *cmd.Context, data any) {
	args := data.(*onlineArgs)

	var onlines []*dao.RoomOnline
	for _, server := range args.Servers {
		onlines = append(onlines, &dao.RoomOnline{
			SubId:  server.Id,
			Online: int(server.Num),
		})
	}
	dao.AddOnline(onlines)
	// log.Debugf("report game online %v", onlines)
}
