// database pool
package dao

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gofishing-plate/internal"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/protobuf/proto"
)

const defaultIdleConns = 100
const defaultOpenConns = 200
const defaultConnLifeTime = 1800 // MySQL默认8小时

var (
	gameDB   *sql.DB // game数据库只读从库
	manageDB *sql.DB // manage数据库主库
)

func init() {
	t := internal.Config().SlaveDataSource
	gameDB, _ = connectDB(t.User, t.Password, t.Addr, t.Name)
	if n := t.MaxIdleConns; n > 0 {
		gameDB.SetMaxIdleConns(n)
	}
	if n := t.MaxOpenConns; n > 0 {
		gameDB.SetMaxOpenConns(n)
	}

	t = internal.Config().ManageDataSource
	manageDB, _ = connectDB(t.User, t.Password, t.Addr, t.Name)
	if n := t.MaxIdleConns; n > 0 {
		manageDB.SetMaxIdleConns(n)
	}
	if n := t.MaxOpenConns; n > 0 {
		manageDB.SetMaxOpenConns(n)
	}
}

func connectDB(user, password, addr, dbName string) (*sql.DB, error) {
	s := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&loc=Local", user, password, addr, dbName)
	db, err := sql.Open("mysql", s)
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(defaultIdleConns)
	db.SetMaxOpenConns(defaultOpenConns)
	db.SetConnMaxLifetime(defaultConnLifeTime * time.Second)
	return db, nil
}

type columnValue struct {
	v   any
	typ string
}

func (col *columnValue) Scan(value any) error {
	if value == nil {
		return nil
	}

	buf, ok := value.([]byte)
	if !ok {
		return errors.New("expect data type []byte")
	}
	switch col.typ {
	case "json":
		return json.Unmarshal(buf, col.v)
	case "pb":
		v := col.v.(proto.Message)
		return proto.Unmarshal(buf, v)
	}
	return errors.New("unknow data type")
}

func (col *columnValue) Value() (driver.Value, error) {
	switch col.typ {
	case "json":
		return json.Marshal(col.v)
	case "pb":
		v := col.v.(proto.Message)
		return proto.Marshal(v)
	}
	return nil, errors.New("unknow data type")
}

func JSON(v any) *columnValue {
	return &columnValue{v: v, typ: "json"}
}

func PB(v any) *columnValue {
	return &columnValue{v: v, typ: "pb"}
}
