package dao

import (
	"database/sql"
	"strings"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type ItemRow struct {
	ShopID    int
	ShopTitle string
}

func GetConfigTableItem() []*ItemRow {
	var rows []*ItemRow
	for _, rowId := range config.Rows("item") {
		var itemId int
		var itemName string
		config.Scan("item", rowId, "ShopID,ShopTitle", &itemId, &itemName)
		rows = append(rows, &ItemRow{ShopID: itemId, ShopTitle: itemName})
	}
	return rows
}

type ItemLogRow struct {
	Way  string
	Name string
}

func GetConfigTableItemLog() []*ItemLogRow {
	var rows []*ItemLogRow
	for _, rowId := range config.Rows("item_log") {
		var way, name string
		config.Scan("item_log", rowId, "Way,Name", &way, &name)
		rows = append(rows, &ItemLogRow{Way: way, Name: name})
	}
	return rows
}

func QueryConfigTable(name string) (*ConfigTable, error) {
	var content string

	err := manageDB.QueryRow("select `content` from gm_table where `name`=?", name).Scan(&content)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &ConfigTable{
		Name:    name,
		Content: content,
		Table:   splitConfigTable(content),
	}, nil
}

type ConfigTable struct {
	Name    string
	Content string
	Table   [][]string
}

func splitConfigTable(content string) [][]string {
	var cells [][]string

	content = strings.ReplaceAll(content, "\r\n", "\n")
	for _, line := range strings.Split(content, "\n") {
		cells = append(cells, strings.Split(line, "\t"))
	}
	return cells
}

// 保存配置表
func UpdateConfigTable(name, content string) error {
	log.Infof("save table %s content %s", name, content)
	manageDB.Exec("insert ignore gm_table(`name`,`content`,`comment`) values(?,?,?)", name, content, "")
	manageDB.Exec("update gm_table set `content`=? where `name`=?", content, name)
	manageDB.Exec("insert into config_table_log(`name`,`content`,`comment`) values(?,?,?)", name, content, "")

	return nil
}

func QueryAllConfigTable() ([]ConfigTable, error) {
	var tables []ConfigTable

	rs, _ := manageDB.Query("select `name`,content from gm_table")
	for rs != nil && rs.Next() {
		var table ConfigTable
		rs.Scan(&table.Name, &table.Content)
		table.Table = splitConfigTable(table.Content)
		tables = append(tables, table)
	}
	return tables, nil
}
