package api

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gofishing-plate/dao"
	"gofishing-plate/internal"

	"github.com/gin-gonic/gin"
	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
)

var errHTTPProxy = errors.New("error http proxy")

type googlePayArgs struct {
	PackageName   string
	ProductId     string
	PruchaseToken string
	Price         float64
}

func createOrderId() string {
	now := time.Now()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d%03d%04d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		now.UnixNano()/1000_000%1000, rand.Intn(10000),
	)
}

func addGoogleOrder(c *Context, data any) (any, error) {
	args := data.(*googlePayArgs)
	purchasesProducts, err := internal.QueryProductPurchase(args.PackageName, args.ProductId, args.PruchaseToken)
	if err != nil {
		return nil, err
	}
	// 订单已取消或中止
	if purchasesProducts.PurchaseState != 0 {
		return nil, errors.New("order is canceled or pending")
	}

	order := &dao.PayOrder{
		OrderId:          purchasesProducts.OrderId,
		Result:           dao.PayOrderNew,
		ExchangeCurrency: purchasesProducts.RegionCode,
		ExchangePrice:    args.Price,
		PaySDK:           "google",
	}
	// v4,{uid},{shop_id},{chan_id}
	params := strings.Split(purchasesProducts.ObfuscatedExternalAccountId+",,,", ",")
	order.BuyUid, _ = strconv.Atoi(params[1])
	order.ItemId, _ = strconv.Atoi(params[2])
	order.ChanId = params[3]
	if purchasesProducts.PurchaseType != nil && *purchasesProducts.PurchaseType == 0 {
		order.Result = dao.PayOrderTest
	}
	if err := addPayOrder(order); err != nil {
		return nil, err
	}
	return cmd.M{"Code": 0}, nil
}

// 测试支付
func addTestOrder(c *Context) {
	uidStr, _ := c.GetQuery("uid")
	itemStr, _ := c.GetQuery("shopId")
	testStr, _ := c.GetQuery("test")

	if itemStr == "" {
		itemStr, _ = c.GetQuery("itemId")
	}

	uid, _ := strconv.Atoi(uidStr)
	itemId, _ := strconv.Atoi(itemStr)
	test, _ := strconv.Atoi(testStr)
	result := dao.PayOrderFinish
	if test != 0 {
		result = dao.PayOrderTest
	}
	order := &dao.PayOrder{
		OrderId:          createOrderId(),
		Result:           int32(result),
		ExchangeCurrency: "USD",
		PaySDK:           "test",
		ItemId:           int(itemId),
		BuyUid:           int(uid),
	}
	e := addPayOrder(order)
	if e != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"Code": e,
			"Msg":  e.Error(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Code": 0,
			"Msg":  "success",
		})
	}
}

// 完成订单。增加订单并发放奖励
func addPayOrder(order *dao.PayOrder) error {
	price, _ := config.Float("Shop", order.ItemId, "ShopsPrice")
	order.Price = price
	order.ExchangePrice = price
	order.Currency = "USD"
	if order.PaySDK == "test" && !internal.Config().IsTest() {
		return errors.New("test env cannnot use test pay")
	}

	userInfo, err := dao.GetRegUserInfo(order.BuyUid)
	if err != nil {
		return err
	}
	order.ChanId = userInfo.ChanId
	err = dao.AddPayOrder(order)
	if err != nil {
		return err
	}
	order.IsFirst, err = dao.FinishPayOrder(order.OrderId, int(order.Result))
	if err != nil {
		return err
	}

	SendMsg(int(userInfo.Uid), "FUNC_Pay", cmd.M{
		"Uid":        int(userInfo.Uid),
		"Price":      price,
		"ItemId":     order.ItemId,
		"ItemNum":    1,
		"Vip":        userInfo.VIP,
		"IsTest":     order.Result == dao.PayOrderTest,
		"OrderId":    order.OrderId,
		"PaySDK":     order.PaySDK,
		"IsFirstPay": order.IsFirst,
	})

	return nil
}

func InitAndroidPublisherService(ctx context.Context) {
	if internal.Config().GoogleAPIs.ServiceAccount != "" {
		internal.InitAndroidPublisherService(ctx, internal.Config().GoogleAPIs.ServiceAccount)
	}
}

func notifyPurchaseSubscription(packageName, productId, purchaseToken string, data *internal.SimpleSubscriptionPurchase) {
	// {uid}:{item_id}:{env}
	params := strings.Split(data.DevelopPayload+":::", ":")

	uid, _ := strconv.Atoi(params[0])
	itemId, _ := strconv.Atoi(params[1])
	price, _ := config.Float("Shop", itemId, "ShopsPrice")

	if data.SubOrderId == "" {
		data.SubOrderId = data.OrderId
	}

	dao.NotifyPurchaseSubscriptionOrder(&dao.SubscriptionOrder{
		Uid:           uid,
		PurchaseToken: purchaseToken,
		PackageName:   packageName,
		ProductId:     productId,
		ExpireMillis:  data.ExpiryTimeMillis,
		OrderId:       data.SubOrderId,
		Price:         price,
	})
	SendMsg(uid, "FUNC_UpdatePurchaseSubcription", cmd.M{
		"Uid":      uid,
		"ExpireTs": data.ExpiryTimeMillis / 1000,
	})

	orderStatus := dao.PayOrderFinish
	if data.IsTest {
		orderStatus = dao.PayOrderTest
	}
	addPayOrder(&dao.PayOrder{
		OrderId:          data.OrderId,
		Result:           int32(orderStatus),
		ExchangeCurrency: "USD",
		ExchangePrice:    price,
		PaySDK:           data.PaySDK,
		ItemId:           int(itemId),
		BuyUid:           int(uid),
	})
}

func PullAndAckPubsub(ctx context.Context) {
	pubsubConfig := internal.Config().Pubsub
	if pubsubConfig.ProjectId != "" {
		internal.PullAndAckSubscription(
			ctx, pubsubConfig.ServiceAccount, pubsubConfig.ProjectId, pubsubConfig.SubscriptionId, notifyPurchaseSubscription,
		)
	}
}
