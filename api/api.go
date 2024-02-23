package api

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"

	"gofishing-plate/internal"

	"github.com/gin-gonic/gin"
	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type Context = gin.Context
type IRoutes = gin.IRoutes

type M map[string]any

type errorCmd struct {
	Id string
}

func (err errorCmd) Error() string {
	return "cmd:" + err.Id + " error"
}

type Handler func(*Context, any) (any, error)

type apiEntry struct {
	h     Handler
	typ   reflect.Type
	codec packageCodec
	// auth   gin.HandlerFunc
}

var apiEntries sync.Map

type packageCodec interface {
	Encode(any) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

type apiPackageCodec struct{}

func (codec *apiPackageCodec) Encode(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (codec *apiPackageCodec) Decode(buf []byte) ([]byte, error) {
	return buf, nil
}

type cmdPackageCodec struct{}

func (codec *cmdPackageCodec) Encode(data any) ([]byte, error) {
	return cmd.Encode("", data)
}

func (codec *cmdPackageCodec) Decode(buf []byte) ([]byte, error) {
	pkg, err := cmd.Decode(buf)
	// 测试服万能签名12345678
	if err != nil && !(err == cmd.ErrInvalidSign && internal.Config().IsTest() && pkg.Sign == "12345678") {
		return nil, err
	}
	return pkg.Data, nil
}

func handleCmd(name string, h Handler, i any) {
	apiEntries.Store(name, &apiEntry{h: h, typ: reflect.TypeOf(i), codec: &cmdPackageCodec{}})
}

func HandleAPI(name string, h Handler, i any) {
	apiEntries.Store(name, &apiEntry{h: h, typ: reflect.TypeOf(i), codec: &apiPackageCodec{}})
}

func matchAPI(c *Context, id string) ([]byte, error) {
	body, _ := c.Get("body")
	rawData, _ := body.([]byte)
	entry, ok := apiEntries.Load(id)
	if !ok {
		return nil, errors.New("dispatch handler: " + id + " is not existed")
	}

	api, _ := entry.(*apiEntry)
	data, err := api.codec.Decode(rawData)
	if err != nil {
		return nil, err
	}

	args := reflect.New(api.typ.Elem()).Interface()
	if err := json.Unmarshal(data, args); err != nil {
		return nil, err
	}
	resp, err := api.h(c, args)

	var errCmd errorCmd
	if errors.As(err, &errCmd) {
		return matchAPI(c, errCmd.Id)
	}
	if err != nil {
		return nil, err
	}
	return api.codec.Encode(resp)
}

// 处理游戏内请求
func dispatchAPI(c *Context) {
	rawData, _ := c.GetRawData() // 只能读一次
	c.Set("body", rawData)
	log.Debugf("recv request %s body %s", c.Request.RequestURI, rawData)

	buf, err := matchAPI(c, c.Request.RequestURI)
	if err != nil {
		buf, _ = json.Marshal(map[string]any{"code": 1, "msg": err.Error()})
		log.Warnf("dispatch api error: %v", err)
	}
	c.Data(200, "application/json", buf)
}

func Run(addr string) {
	r := gin.Default()
	r.Use(func(c *Context) {
		c.Writer.Header().Add("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Add("Access-Control-Allow-Methods", "GET,HEAD,PUT,POST,DELETE,PATCH,OPTIONS")
	})

	if !internal.Config().IsTest() {
		gin.SetMode(gin.ReleaseMode)
	}

	for _, preload := range preloadRoutes {
		preload(r)
	}
	apiEntries.Range(func(key, value any) bool {
		r.POST(key.(string), dispatchAPI)
		return true
	})
	r.Static("/"+internal.Config().ResourcePath, internal.Config().ResourcePath)

	if err := r.Run(addr); err != nil {
		log.Fatalf("start gin server fail, %v", err)
	}
}
