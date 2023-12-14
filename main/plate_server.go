//go:build ignore
// +build ignore

package main

import (
	"context"
	"flag"
	"fmt"
	"runtime"

	"gofishing-plate/api"
	_ "gofishing-plate/api/gm"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

var (
	port       = flag.String("port", "8001", "server port")
	serverName = flag.String("name", "plate", "server name")
)

func main() {
	flag.Parse()

	addr := fmt.Sprintf(":%s", *port)
	log.Infof("start plate server, listen %s", *port)
	api.InitAndroidPublisherService(context.Background())
	api.LoadRemoteTables()

	go func() { api.Run(addr) }()
	go func() { api.PullAndAckPubsub(context.Background()) }()
	cmd.RegisterService(&cmd.ServiceConfig{Name: *serverName, Addr: addr})

	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Error(err)
			log.Errorf("%s", buf)
		}
	}()

	for {
		// handle message
		cmd.RunOnce()
		util.GetTimerSet().RunOnce()
	}
}
