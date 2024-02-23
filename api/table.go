package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

var resourceTableDir string
var matchTableRe = regexp.MustCompile(`^[A-Za-z0-9_]+.tbl$`)

func init() {
	resourceTableDir = internal.Config().ResourcePath + "/tables"
	compressConfigTables()

	cmd.Bind("func_effectConfigTable", funcEffectConfigTable, (*tableArgs)(nil)).SetNoQueue()
}

// 加载配置
func LoadRemoteTables() {
	tables, err := dao.QueryAllConfigTable()
	if err != nil {
		log.Fatalf("load all table config %v", err)
	}
	for _, table := range tables {
		err = config.LoadTable(table.Name, []byte(table.Content))
		if err != nil {
			log.Fatalf("load table config %v", err)
		}
	}
}

type tableArgs struct {
	Name   string
	Tables []string
}

func funcEffectConfigTable(ctx *cmd.Context, data any) {
	args := data.(*tableArgs)
	for _, name := range args.Tables {
		table, err := dao.QueryConfigTable(name)
		if err != nil {
			log.Errorf("load table %s error: %v", name, err)
			return
		}

		log.Info("effect config table", name)
		log.Info(table.Content)
		config.LoadTable(name, []byte(table.Content))
	}
}

func saveConfigTables(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	files := map[string]string{}
	for _, f := range r.File {
		ext := filepath.Ext(f.Name)
		if !matchTableRe.MatchString(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		buf, err := io.ReadAll(rc)
		if err != nil {
			return err
		}
		rc.Close()

		base := filepath.Base(f.Name)
		name := base[:len(base)-len(ext)]
		files[name] = string(buf)
	}

	for name, content := range files {
		// 检测配置文件格式
		if err := config.ValidateConfigTable([]byte(content)); err != nil {
			return fmt.Errorf("config %s format error: \n%v", name, err.Error())
		}
	}

	var tables []string
	for name, content := range files {
		dao.UpdateConfigTable(name, content)
		tables = append(tables, name)
	}
	log.Debugf("upload config tables %s success", strings.Join(tables, ","))
	cmd.Forward("*", "FUNC_EffectConfigTable", cmd.M{"Tables": tables})
	if err := compressConfigTables(); err != nil {
		log.Errorf("compress config tables fail: %v", err)
	}
	return nil
}

// 获取配置文件下载路径
func getNewestConfigTableUrl() string {
	var url string
	for i := 0; i < 2; i++ {
		var tables []os.FileInfo
		filepath.Walk(resourceTableDir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if ext := filepath.Ext(info.Name()); ext != ".zip" {
				return nil
			}
			tables = append(tables, info)
			return nil
		})
		sort.Slice(tables, func(i, j int) bool {
			return tables[i].ModTime().After(tables[j].ModTime())
		})
		// 删除24h前的旧配置，最新配置不可删除
		for i := 1; i < len(tables); i++ {
			info := tables[i]
			if time.Since(info.ModTime()) > 24*time.Hour {
				os.Remove(resourceTableDir + "/" + info.Name())
			}
		}

		if len(tables) > 0 {
			url = fmt.Sprintf("%s/%s/%s", internal.Config().ResourceURL, resourceTableDir, tables[0].Name())
		}
		if url != "" {
			break
		}
		if err := compressConfigTables(); err != nil {
			log.Errorf("获取配置文件下载路径 %v", err)
			return ""
		}
	}

	return url
}

// 1、数据库加载最新配置
// 2、tbl格式转成json
// 3、压缩成zip文件
func compressConfigTables() error {
	tmpDir := "/tmp/" + util.GUID()
	os.MkdirAll(tmpDir, 0755)

	tables, err := dao.QueryAllConfigTable()
	if err != nil {
		return err
	}
	for _, table := range tables {
		clientTable := config.ExportConfigTable([]byte(table.Content))
		buf, _ := json.Marshal(cmd.M{
			"Table": json.RawMessage(clientTable),
			"Name":  table.Name,
		})
		os.WriteFile(tmpDir+"/"+table.Name+".json", buf, 0644)
	}

	zipFile, _ := os.Create(resourceTableDir + "/" + createFileName() + ".zip")
	defer zipFile.Close()
	w := zip.NewWriter(zipFile)
	defer w.Close()
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, lastErr error) error {
		if info.IsDir() {
			return nil
		}

		f, err := w.Create(info.Name())
		if err != nil {
			return err
		}
		buf, _ := os.ReadFile(path)
		_, err = f.Write(buf)
		if err != nil {
			return err
		}
		return nil
	})
	return os.RemoveAll(tmpDir)
}

// POST file 上传配置表
func uploadConfigTables(c *Context) {
	path := internal.Config().ResourcePath + "/" + util.GUID() + ".zip"
	file, _ := c.FormFile("from_svn")
	if err := c.SaveUploadedFile(file, path); err != nil {
		log.Errorf("save upload file fail: %v", err)
	}
	err := saveConfigTables(path)
	os.Remove(path)

	httpStatus, response := 200, M{"code": 0, "msg": "ok"}
	if err != nil {
		httpStatus, response = 500, M{"code": 1, "msg": err.Error()}
	}
	c.PureJSON(httpStatus, response)
}
