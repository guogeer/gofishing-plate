package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/guogeer/quasar/config"
)

const (
	PayOrderNew    = 1 // 新建订单
	PayOrderFinish = 3 // 完成订单
	PayOrderTest   = 4 // 测试订单
)

const exchangeRates = "SG; 520000000; US; 374990000; HK; 2905000000; MO; 3000000000; CA; 498000000; CN; 549000000; TH; 12517000000"

type PayOrderParams struct {
	SubId          int
	NumAfterPay    float64
	LevelBeforePay int
}

type PayOrder struct {
	Params           PayOrderParams `json:"params,omitempty"`
	OrderId          string         `json:"orderId,omitempty"`          // 订单ID
	BuyUid           int            `json:"buyUid,omitempty"`           // 购买ID
	ItemId           int            `json:"itemId,omitempty"`           // 商品ID
	ExchangeCurrency string         `json:"exchangeCurrency,omitempty"` // 当地货币类型，如HKD
	ExchangePrice    float64        `json:"exchangePrice,omitempty"`    // 当地价格
	CreateTime       string         `json:"createTime,omitempty"`
	ChanId           string         `json:"chanId,omitempty"`
	PaySDK           string         `json:"paySDK,omitempty"`
	Game             string         `json:"game,omitempty"`
	Result           int32          `json:"result,omitempty"`
	Price            float64        `json:"price,omitempty"`
	Currency         string         `json:"currency,omitempty"`
	IsFirst          bool           `json:"isFirst,omitempty"`
	ItemName         string         `json:"itemName,omitempty"`
	RoomName         string         `json:"roomName,omitempty"`
	CountPrice       string         `json:"countPrice,omitempty"`
}

func AddPayOrder(order *PayOrder) error {
	rs, err := manageDB.Exec("insert ignore into charge_order(order_id,game,pay_sdk,chan_id,buy_uid,item_id,item_num,currency,rmb,exchange_currency,exchange_price,result,params,create_time) values(?,?,?,?,?,?,?,?,?,?,?,?,?,now())",
		order.OrderId, order.Game, order.PaySDK, order.ChanId, order.BuyUid, order.ItemId, 1, order.Currency, order.Price, order.ExchangeCurrency, order.ExchangePrice, 1, JSON(order.Params))
	if err != nil {
		return err
	}
	if num, _ := rs.RowsAffected(); num == 0 {
		return errors.New("order is existed or invalid")
	}
	return nil
}

func FinishPayOrder(orderId string, result int) (bool, error) {
	if result != PayOrderTest {
		result = PayOrderFinish
	}
	params := &PayOrderParams{}

	var first, uid int32
	manageDB.QueryRow("select buy_uid,params from charge_order where order_id=?", orderId).Scan(&uid, JSON(params))
	params.SubId = 101

	manageDB.QueryRow("select 1 from charge_order where buy_uid=? and result>=?", uid, PayOrderFinish).Scan(&first)
	first ^= 1
	manageDB.Exec("update charge_order set first_pay=?,result=?,notify_time=now(),params=? where order_id=?", first, result, JSON(params), orderId)
	return first != 0, nil
}

// 查询订单
func QueryPayOrder(orderId string, uid int, result, timeRange []string, current, pageSize int) ([]*PayOrder, int, string, error) {
	var params []any

	where := " where 1=1"
	if orderId != "" {
		where = where + " and order_id=?"
		params = append(params, orderId)
	}
	if uid > 0 {
		where = where + " and buy_uid=?"
		params = append(params, uid)
	}
	if len(timeRange) > 1 {
		where = where + " and create_time between ? and ?"
		params = append(params, timeRange[0], timeRange[1])
	}
	if len(result) > 0 {
		where = where + fmt.Sprintf(" and result in (%s)", strings.Join(result, ","))
	}

	var totalRows int
	var totalPrice float64
	manageDB.QueryRow("select sum(rmb),count(*) from charge_order"+where, params...).Scan(&totalPrice, &totalRows)

	params = append(params, (current-1)*pageSize, pageSize)
	rs, err := manageDB.Query("select order_id,buy_uid,chan_id,item_id,exchange_currency,exchange_price,currency,rmb,result,pay_sdk,params,create_time from charge_order"+where+" order by id desc limit ?,?", params...)
	if err != nil {
		return nil, 0, "", err
	}

	rates := map[string]int{}
	exchangeRateArr := strings.Split(exchangeRates, "; ")
	for i := 0; 2*i+1 < len(exchangeRateArr); i++ {
		region, rate := exchangeRateArr[2*i], exchangeRateArr[2*i+1]
		rates[region], _ = strconv.Atoi(rate)
	}

	var orders []*PayOrder
	for rs.Next() {
		order := &PayOrder{}
		err := rs.Scan(&order.OrderId, &order.BuyUid, &order.ChanId, &order.ItemId, &order.ExchangeCurrency, &order.ExchangePrice, &order.Currency, &order.Price, &order.Result, &order.PaySDK, JSON(&order.Params), &order.CreateTime)
		if err != nil {
			return nil, 0, "", err
		}

		order.RoomName, _ = config.String("room", order.Params.SubId, "roomName")
		order.ItemName, _ = config.String("shop", order.ItemId, "shopTitle")

		countPrice := "汇率缺失"
		currency, exchangeCurrency := order.Currency, order.ExchangeCurrency
		if len(currency) > 2 {
			currency = currency[:2]
		}
		if len(exchangeCurrency) > 2 {
			exchangeCurrency = exchangeCurrency[:2]
		}
		if rates[currency] != 0 && rates[exchangeCurrency] != 0 {
			price := float64(rates[currency]) / float64(rates[exchangeCurrency]) * float64(order.ExchangePrice)
			countPrice = fmt.Sprintf("%s%.2f", order.Currency, price)
		}

		order.CountPrice = countPrice
		orders = append(orders, order)
	}
	return orders, totalRows, fmt.Sprintf("USD%.2f", totalPrice), nil
}

type SubscriptionOrder struct {
	Id            int
	Uid           int
	OrderId       string
	PurchaseToken string
	ProductId     string
	PackageName   string
	DeferDays     int
	ExpireMillis  int64
	Price         float64
}

func DeferPurchaseSubscriptionOrder() (*SubscriptionOrder, error) {
	order := &SubscriptionOrder{}

	tx, _ := manageDB.Begin()
	err := tx.QueryRow("select id,buy_uid,defer_days,purchase_token,product_id,package_name,expire_millis from charge_subscription where defer_days>0 limit 1 for update").Scan(
		&order.Id, &order.Uid, &order.DeferDays, &order.PurchaseToken, &order.ProductId, &order.PackageName, &order.ExpireMillis)
	if err != sql.ErrNoRows {
		tx.Exec("update charge_subscription set defer_days=defer_days-(?) where id=?", order.DeferDays, order.Id)
	}
	tx.Commit()
	return order, nil
}

func UpdatePurchaseSubscriptionOrderExpiryTime(id int, expiryTimeMillis int64) error {
	_, err := manageDB.Exec("update charge_subscription set expire_millis=? where id=?", expiryTimeMillis, id)
	return err
}

func NotifyPurchaseSubscriptionOrder(order *SubscriptionOrder) error {
	manageDB.Exec("delete from charge_subscription where buy_uid=?", order.Uid)
	manageDB.Exec("insert into charge_subscription(buy_uid,order_id,purchase_token,product_id,package_name,price,create_time) values(?,?,?,?,?,?,now())",
		order.Uid, order.OrderId, order.PurchaseToken, order.ProductId, order.PackageName, order.Price)
	return nil
}

func QueryPurchaseSubscriptionOrder(orderId string) (*SubscriptionOrder, error) {
	var order SubscriptionOrder
	manageDB.QueryRow("select id,buy_uid,order_id,product_id,price from charge_subscription where order_id=?", orderId).Scan(
		&order.Id, &order.Uid, &order.OrderId, &order.ProductId, &order.Price,
	)
	if order.OrderId == "" {
		return nil, errors.New("empty order")
	}
	return &order, nil
}
