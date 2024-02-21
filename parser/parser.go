package parser

import (
	"context"
	"fmt"
	"jtso/logger"
	"jtso/sqlite"
	"time"

	"github.com/openconfig/gnmic/pkg/api"
	"github.com/openconfig/gnmic/pkg/formatters"
)

var root *TreeNode
var global []string

func LaunchSearch(h string, p string, m bool) (string, *TreeNode) {

	logger.Log.Infof("Start subscription for router %s and xpath %s", h, p)

	// Init global variable
	root = NewTree("", map[string]interface{}{})
	global = make([]string, 0)

	// create a target
	tg, err := api.NewTarget(
		api.Name("jtso"),
		api.Address(h),
		api.Username(sqlite.ActiveCred.GnmiUser),
		api.Password(sqlite.ActiveCred.GnmiPwd),
		api.SkipVerify(true),
		api.Insecure(true),
	)
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI target: %v", err)
		return "TARGET_KO", nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = tg.CreateGNMIClient(ctx)
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI client: %v", err)
		return "CLIENT_KO", nil
	}
	defer tg.Close()
	// create a gNMI subscribeRequest
	subReq, err := api.NewSubscribeRequest(
		api.Encoding("proto"),
		api.SubscriptionListMode("stream"),
		api.Subscription(
			api.Path(p),
			api.SubscriptionMode("sample"),
			api.SampleInterval(30*time.Second),
		))
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI subscription: %v", err)
		return "SUB_KO", nil
	}

	go tg.Subscribe(ctx, subReq, "sub1")

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(60 * time.Second):
			tg.StopSubscription("sub1")

		}
	}()
	subRspChan, subErrChan := tg.ReadSubscriptions()
	for {
		select {
		case rsp := <-subRspChan:
			r, _ := formatters.ResponsesFlat(rsp.Response)
			for k, v := range r {
				parseXpath(k, fmt.Sprint(v), m)
			}

		case tgErr := <-subErrChan:
			//traverseTree(root)
			logger.Log.Infof("End of the subscription after timeout: %v", tgErr)
			return "OK", root
		}
	}

}
