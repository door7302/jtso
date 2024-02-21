package parser

import (
	"context"
	"fmt"
	"jtso/logger"
	"jtso/sqlite"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/openconfig/gnmic/pkg/api"
	"github.com/openconfig/gnmic/pkg/formatters"
)

var root *TreeNode
var global []string
var re1, re2 *regexp.Regexp

var StreamObj *Streamer

type Streamer struct {
	Stream        bool
	Path          string
	Router        string
	Port          int
	Merger        bool
	Status        string
	Result        *TreeNode
	Context       echo.Context
	StopStreaming chan struct{}
}

func init() {
	// init re
	re1 = regexp.MustCompile("(\\d+)")
	re2 = regexp.MustCompile("(.*)\\[(.*)=(.*)\\]")

	// init streamer
	StreamObj = new(Streamer)
}

func streamData(m string, s string) {
	data := map[string]interface{}{
		"msg":    m,
		"status": s,
	}
	err := StreamObj.Context.JSON(http.StatusOK, data)
	if err != nil {
		logger.Log.Errorf("Error during streamin data: %v", err)
	}
	StreamObj.Context.Response().Flush()
}

func advancedSplit(path string) []string {
	if strings.Contains(path, "=") && strings.Contains(path, "[") {
		var newPath string
		escape := false

		for _, w := range path {
			if w == '[' {
				escape = true
			}
			if w == ']' {
				escape = false
			}
			if !escape {
				if w == '/' {
					newPath += "£££"
				} else {
					newPath += string(w)
				}
			} else {
				newPath += string(w)
			}
		}
		return strings.Split(newPath, "£££")
	}
	return strings.Split(path, "/")
}

func printTree(node map[string]interface{}, indent int, o map[string]interface{}) {
	for k, v := range node {
		if reflect.TypeOf(v).Kind() == reflect.Map {
			fmt.Printf("%s+ %s\n", strings.Repeat("  ", indent), k)
			o[k] = map[string]interface{}{}
			printTree(v.(map[string]interface{}), indent+1, o[k].(map[string]interface{}))
		} else {
			o[k] = v
			fmt.Printf("%s+ %s: %s\n", strings.Repeat("  ", indent), k, fmt.Sprint(v))
		}
	}

}

func traverseTree(node *TreeNode) {
	global = append(global, node.Data.(string))
	if len(node.Children) != 0 {
		for _, child := range node.Children {
			traverseTree(child)
		}
		global = global[:len(global)-1]
	} else {
		path := strings.Join(global, "/")
		fmt.Printf("%s\n", path)
		output := make(map[string]interface{})
		output[path] = make(map[string]interface{})
		printTree(node.Value, 1, output[path].(map[string]interface{}))
		global = global[:len(global)-1]
	}
}

func parseXpath(xpath string, value string, merge bool) error {

	var parent *TreeNode
	var key []string
	var val map[string]interface{}

	key = make([]string, 0)

	if merge {
		xpath = re1.ReplaceAllString(xpath, "x")
	}
	fmt.Println(xpath)
	lpath := advancedSplit(xpath)

	parent = root
	for i, v := range lpath {
		if i == len(lpath)-1 {
			if len(key) == 0 {
				val["alone"] = value
			} else {
				val = make(map[string]interface{})
				tmp := val
				for ki, kv := range key {
					if ki == len(key)-1 {
						tmp[kv] = value
					} else {
						tmp[kv] = make(map[string]interface{})
						tmp = tmp[kv].(map[string]interface{})
					}
				}
			}
		} else {
			val = make(map[string]interface{})
		}
		if strings.Contains(v, "=") {
			matches := re2.FindStringSubmatch(v)

			composite := matches[1] + "[" + matches[2] + "=*]"
			node, result := parent.FindNode(composite)
			if result {
				node.AddValue(val)
			} else {
				node = parent.InsertChild(composite, val)
			}
			parent = node
			key = append(key, matches[3])
		} else {
			node, result := parent.FindNode(v)
			if result {
				node.AddValue(val)
			} else {
				node = parent.InsertChild(v, val)
			}
			parent = node
		}
	}
	return nil
}

func LaunchSearch() {

	logger.Log.Infof("Start subscription for router %s and xpath %s", StreamObj.Router, StreamObj.Path)
	streamData(fmt.Sprintf("Start subscription for router %s and xpath %s", StreamObj.Router, StreamObj.Path), "OK")

	// Init global variable
	root = NewTree("", map[string]interface{}{})
	global = make([]string, 0)

	// create a target
	tg, err := api.NewTarget(
		api.Name("jtso"),
		api.Address(StreamObj.Router+":"+fmt.Sprint(StreamObj.Port)),
		api.Username(sqlite.ActiveCred.GnmiUser),
		api.Password(sqlite.ActiveCred.GnmiPwd),
		api.SkipVerify(true),
		api.Insecure(true),
	)
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI target: %v", err)
		StreamObj.Status = "TARGET_KO"
		close(StreamObj.StopStreaming)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = tg.CreateGNMIClient(ctx)
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI client: %v", err)
		StreamObj.Status = "CLIENT_KO"
		close(StreamObj.StopStreaming)
		return
	}

	defer tg.Close()
	// create a gNMI subscribeRequest
	subReq, err := api.NewSubscribeRequest(
		api.Encoding("proto"),
		api.SubscriptionListMode("stream"),
		api.Subscription(
			api.Path(StreamObj.Path),
			api.SubscriptionMode("sample"),
			api.SampleInterval(30*time.Second),
		))
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI subscription: %v", err)
		StreamObj.Status = "SUB_KO"
		close(StreamObj.StopStreaming)
		return
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
				parseXpath(k, fmt.Sprint(v), StreamObj.Merger)
			}

		case tgErr := <-subErrChan:
			//traverseTree(root)
			logger.Log.Infof("End of the subscription after timeout: %v", tgErr)
			StreamObj.Result = root
			StreamObj.Status = "END_OK"
			close(StreamObj.StopStreaming)
			return
		}
	}

}
