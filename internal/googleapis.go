package internal

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/guogeer/quasar/log"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	ErrInvalidRequest       = errors.New("invalid request")
	androidpublisherService *androidpublisher.Service
)

type pubsubNotification struct {
	Version         string `json:"version"`
	PackageName     string `json:"packageName"`
	EventTimeMillis string `json:"eventTimeMillis"`

	SubscriptionNotification struct {
		Version          string `json:"version"`
		NotificationType int    `json:"notificationType"`
		PurchaseToken    string `json:"purchaseToken"`
		SubscriptionId   string `json:"subscriptionId"`
	} `json:"subscriptionNotification"`

	OneTimeProductNotification struct {
		Version          string `json:"version"`
		NotificationType int    `json:"notificationType"`
		PurchaseToken    string `json:"purchaseToken"`
		Sku              string `json:"sku"`
	} `json:"oneTimeProductNotification"`

	TestNotification struct {
		Version string `json:"version"`
	} `json:"testNotification"`
}

type SubscriptionPurchase = androidpublisher.SubscriptionPurchase
type PubsubNotifyFunc func(string, string, string, *SimpleSubscriptionPurchase)

type SimpleSubscriptionPurchase struct {
	ExpiryTimeMillis int64
	DevelopPayload   string
	OrderId          string
	IsTest           bool
	PaySDK           string
	SubOrderId       string
}

func QueryPurchaseSubscription(packageName, productId, purchaseToken string) (*androidpublisher.SubscriptionPurchase, error) {
	purchasesSubscriptionsService := androidpublisher.NewPurchasesSubscriptionsService(androidpublisherService)
	return purchasesSubscriptionsService.Get(packageName, productId, purchaseToken).Do()
}

func DeferPurchaseSubscription(packageName, productId, purchaseToken string, desiredExpiryTimeMillis, expectedExpiryTimeMillis int64) (int64, error) {
	deferRequest := &androidpublisher.SubscriptionPurchasesDeferRequest{
		DeferralInfo: &androidpublisher.SubscriptionDeferralInfo{
			DesiredExpiryTimeMillis:  desiredExpiryTimeMillis,
			ExpectedExpiryTimeMillis: expectedExpiryTimeMillis,
		},
	}
	purchasesSubscriptionsService := androidpublisher.NewPurchasesSubscriptionsService(androidpublisherService)
	subscriptionPurchase, err := purchasesSubscriptionsService.Defer(packageName, productId, purchaseToken, deferRequest).Do()
	if err != nil {
		return 0, err
	}
	return subscriptionPurchase.NewExpiryTimeMillis, nil
}

func QueryProductPurchase(packageName, productId, purchaseToken string) (*androidpublisher.ProductPurchase, error) {
	purchasesProductsService := androidpublisher.NewPurchasesProductsService(androidpublisherService)
	purchasesProducts, err := purchasesProductsService.Get(packageName, productId, purchaseToken).Do()
	if err != nil {
		var googleErr *googleapi.Error
		if errors.As(err, &googleErr) && googleErr.Code == 400 {
			err = ErrInvalidRequest
		}
	}
	return purchasesProducts, err
}

// {"version":"1.0","packageName":"bingo.town.free.wild.journey","eventTimeMillis":"1620805531405","subscriptionNotification":{"version":"1.0","notificationType":13,"purchaseToken":"kcipejgoafhenikebfcpkmjf.AO-J1Oz-A9b5HBEgouwMdjMVE9uyvTgt537Ev1zbgfn8Lp6gaJNaC2WOTU8orUcNaJjMi-3J4ibSTSe-j_GvRRc7V1zxcgW_oPXfJN4ItIz-pQlbCIuP7oo","subscriptionId":"bingo_pid_goldmembership1"}}
func handlePubsubMsg(data []byte, callback PubsubNotifyFunc) error {
	log.Infof("receive pubsub data: %s", data)
	if len(data) == 0 {
		return nil
	}

	msg := &pubsubNotification{}
	err := json.Unmarshal(data, msg)
	if err != nil {
		return err
	}

	notify := msg.SubscriptionNotification
	switch notify.NotificationType {
	default:
		return nil
	// 2 续订了处于活动状态的订阅
	// 4 购买了新的订阅
	case 1, 2, 4:
		// 请求管理后台创建订单
	}
	response, err := QueryPurchaseSubscription(msg.PackageName, notify.SubscriptionId, notify.PurchaseToken)
	if err != nil {
		return err
	}
	// v4,{uid},{item_id},{chan_id},{nettype or env}
	params := strings.Split(response.ObfuscatedExternalAccountId+",,,,", ",")
	if params[0] == "v4" {
		params = params[1:]
	}

	clientEnv, _ := strconv.Atoi(params[3])
	if GetClientEnv() != clientEnv {
		return nil
	}

	simpleSub := &SimpleSubscriptionPurchase{
		ExpiryTimeMillis: response.ExpiryTimeMillis,
		DevelopPayload:   strings.Join(params, ":"),
		OrderId:          response.OrderId,
		IsTest:           response.PurchaseType != nil && *response.PurchaseType == 0,
		PaySDK:           "google",
	}

	if callback != nil {
		callback(msg.PackageName, notify.SubscriptionId, notify.PurchaseToken, simpleSub)
	}
	return nil
}

func InitAndroidPublisherService(ctx context.Context, serviceAccountFile string) {
	opt := option.WithCredentialsFile(serviceAccountFile)
	service, err := androidpublisher.NewService(ctx, opt)
	if err != nil {
		panic(err)
	}
	androidpublisherService = service
}

// 实时开发者通知
func PullAndAckSubscription(ctx context.Context, serviceAccountFile, projectId, SubscriptionId string, callback PubsubNotifyFunc) {
	opt := option.WithCredentialsFile(serviceAccountFile)
	client, err := pubsub.NewClient(ctx, projectId, opt)
	if err != nil {
		log.Fatal(err)
	}
	for {
		// Use a callback to receive messages via subscription1.
		sub := client.Subscription(SubscriptionId)
		err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			handleErr := handlePubsubMsg(m.Data, callback)
			if handleErr != nil {
				log.Errorf("handle pubsub msg error: %v", handleErr)
			}
			m.Ack() // Acknowledge that we've consumed the message.
		})
		if err != nil {
			log.Errorf("receive pubsub message error: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}
