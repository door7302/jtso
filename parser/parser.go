package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"jtso/logger"
	"jtso/sqlite"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openconfig/gnmic/pkg/api"
	"github.com/openconfig/gnmic/pkg/api/target"
	"github.com/openconfig/gnmic/pkg/formatters"
)

const PATH_CERT string = "/var/shared/telegraf/cert/"

var root *TreeNode
var global []string
var re1, re2 *regexp.Regexp
var StreamObj *Streamer

type TreeJs struct {
	Id     string `json:"id"`
	Parent string `json:"parent"`
	Text   string `json:"text"`
	Icon   string `json:"icon"`
}

type Streamer struct {
	Stream        int
	Path          string
	Router        string
	Port          int
	Merger        bool
	Ticker        time.Time
	ForceFlush    bool
	Result        *TreeNode
	Flusher       http.Flusher
	Writer        http.ResponseWriter
	Error         error
	XpathCpt      int
	XpathList     map[string]struct{}
	StopStreaming chan struct{}
}

func genUUID() string {
	return uuid.New().String()
}

func init() {
	// init re
	re1 = regexp.MustCompile("(\\d+)")
	re2 = regexp.MustCompile("(.*)\\[(.*)=(.*)\\]")

	// init streamer
	StreamObj = new(Streamer)
}

func ToJSON(data map[string]interface{}) string {
	// Convert the data map to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return ""
	}
	return string(jsonData)
}

func StreamData(m string, s string, payload ...string) {
	var pl string
	if len(payload) == 0 {
		pl = ""
	} else {
		pl = payload[0]
	}

	data := map[string]interface{}{
		"msg":     m,
		"status":  s,
		"payload": pl,
	}
	jsonData := fmt.Sprintf("data: %s\n\n", ToJSON(data))
	fmt.Fprint(StreamObj.Writer, jsonData)

	// Compute the time between 2 flushs - min must be 1 sec
	elapsedTime := time.Since(StreamObj.Ticker)
	if elapsedTime.Seconds() >= 1.0 || StreamObj.ForceFlush {
		StreamObj.Flusher.Flush()
	}
	StreamObj.Ticker = time.Now()

}

func advancedSplit(path string, merge bool) []string {

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
		lp := strings.Split(newPath, "£££")
		if merge {
			for i, v := range lp {
				if strings.Contains(v, "=") {
					lp[i] = re1.ReplaceAllString(v, "x")
				}
			}
		}
		return lp
	}
	return strings.Split(path, "/")
}

func PrintTree(node map[string]interface{}, indent int, o map[string]interface{}, parentKey string, j *[]TreeJs) {
	var entry TreeJs

	for k, v := range node {
		if reflect.TypeOf(v).Kind() == reflect.Map {
			newkey := genUUID()
			entry = TreeJs{
				Id:     newkey,
				Parent: parentKey,
				Text:   k,
				Icon:   "fas fa-search-plus",
			}
			*j = append(*j, entry)

			//fmt.Printf("%s+ %s\n", strings.Repeat("  ", indent), k)
			o[k] = map[string]interface{}{}
			PrintTree(v.(map[string]interface{}), indent+1, o[k].(map[string]interface{}), newkey, j)
		} else {
			o[k] = v
			//fmt.Printf("%s+ %s: %s\n", strings.Repeat("  ", indent), k, fmt.Sprint(v))
			entry = TreeJs{
				Id:     genUUID(),
				Parent: parentKey,
				Text:   fmt.Sprintf("%s = %s", k, fmt.Sprint(v)),
				Icon:   "fas fa-sign-out-alt",
			}
			*j = append(*j, entry)
		}
	}

}

func TraverseTree(node *TreeNode, parentKey string, j *[]TreeJs) {
	global = append(global, node.Data.(string))

	if len(node.Children) != 0 {
		for _, child := range node.Children {
			TraverseTree(child, parentKey, j)
		}
		global = global[:len(global)-1]
	} else {
		path := strings.Join(global, "/")
		var entry TreeJs
		newkey := genUUID()

		entry = TreeJs{
			Id:     newkey,
			Parent: parentKey,
			Text:   path,
			Icon:   "fas fa-search-plus",
		}
		*j = append(*j, entry)

		//fmt.Printf("%s\n", path)
		output := make(map[string]interface{})
		output[path] = make(map[string]interface{})
		PrintTree(node.Value, 1, output[path].(map[string]interface{}), newkey, j)
		global = global[:len(global)-1]
	}
}

func parseXpath(xpath string, value string, merge bool) error {

	var parent *TreeNode
	var key []string
	var val map[string]interface{}

	key = make([]string, 0)

	lpath := advancedSplit(xpath, merge)
	xpathKey := strings.Join(lpath, "/")
	// increment counter and save the Xpath
	_, ok := StreamObj.XpathList[xpathKey]
	if !ok {
		StreamObj.XpathCpt += 1
		StreamObj.XpathList[strings.Join(lpath, "/")] = struct{}{}
		StreamData(fmt.Sprintf("%d", StreamObj.XpathCpt), "XPATH")
	}
	// Old method to share XPATH
	// StreamData(fmt.Sprintf("XPATH Extracted: %s", strings.Join(lpath, "/")), "OK")
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

func LaunchSearch(timeout time.Duration) {

	logger.Log.Infof("Start subscription for router %s and xpath %s", StreamObj.Router, StreamObj.Path)
	StreamData(fmt.Sprintf("Start subscription for router %s and xpath %s", StreamObj.Router, StreamObj.Path), "OK")

	// Init global variable
	root = NewTree("", map[string]interface{}{})
	global = make([]string, 0)
	var tg *target.Target
	var err error

	// Retrieve cred info
	tls := false
	skip := false
	clienttls := false
	if sqlite.ActiveCred.UseTls == "yes" {
		tls = true
	}
	if sqlite.ActiveCred.SkipVerify == "yes" {
		skip = true
	}
	if sqlite.ActiveCred.ClientTls == "yes" {
		clienttls = true
	}

	if tls {
		if clienttls {
			StreamData("Try to create Target", "OK")
			// create a target
			tg, err = api.NewTarget(
				api.Name("jtso"),
				api.Address(StreamObj.Router+":"+fmt.Sprint(StreamObj.Port)),
				api.Username(sqlite.ActiveCred.GnmiUser),
				api.Password(sqlite.ActiveCred.GnmiPwd),
				api.SkipVerify(skip),
				api.Insecure(false),
				api.TLSCA(PATH_CERT+"RootCA.crt"),
				api.TLSCert(PATH_CERT+"client.crt"),
				api.TLSKey(PATH_CERT+"client.key"),
			)

		} else {
			StreamData("Try to create Target", "OK")
			// create a target
			tg, err = api.NewTarget(
				api.Name("jtso"),
				api.Address(StreamObj.Router+":"+fmt.Sprint(StreamObj.Port)),
				api.Username(sqlite.ActiveCred.GnmiUser),
				api.Password(sqlite.ActiveCred.GnmiPwd),
				api.SkipVerify(skip),
				api.Insecure(false),
				api.TLSCA(PATH_CERT+"RootCA.crt"),
			)

		}
	} else {
		StreamData("Try to create Target", "OK")
		// create a target
		tg, err = api.NewTarget(
			api.Name("jtso"),
			api.Address(StreamObj.Router+":"+fmt.Sprint(StreamObj.Port)),
			api.Username(sqlite.ActiveCred.GnmiUser),
			api.Password(sqlite.ActiveCred.GnmiPwd),
			api.SkipVerify(skip),
			api.Insecure(true),
		)
	}

	if err != nil {
		logger.Log.Errorf("Unable to create gNMI target: %v", err)
		StreamData(fmt.Sprintf("Unable to create gNMI target: %v", err), "ERROR")
		StreamObj.Error = err
		close(StreamObj.StopStreaming)
		return
	}
	StreamData("gNMI Target created", "OK")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = tg.CreateGNMIClient(ctx)
	if err != nil {
		logger.Log.Errorf("Unable to create gNMI client: %v", err)
		StreamData(fmt.Sprintf("Unable to create gNMI client: %v", err), "ERROR")
		StreamObj.Error = err
		close(StreamObj.StopStreaming)
		return
	}
	StreamData("Create gNMI Client", "OK")

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
		StreamData(fmt.Sprintf("Unable to create gNMI subscription: %v", err), "ERROR")
		StreamObj.Error = err
		close(StreamObj.StopStreaming)
		return
	}
	StreamData("Create gNMI Subscription", "OK")

	go tg.Subscribe(ctx, subReq, "sub1")

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeout * time.Second):
			logger.Log.Infof("End of the subscription timer")
			tg.StopSubscription("sub1")
		}
	}()

	subRspChan, subErrChan := tg.ReadSubscriptions()
	StreamData("Start collection data", "OK")
	StreamData("Please wait...", "OK")
	StreamObj.ForceFlush = false
	for {
		select {
		case rsp := <-subRspChan:
			r, _ := formatters.ResponsesFlat(rsp.Response)
			for k, v := range r {
				parseXpath(k, fmt.Sprint(v), StreamObj.Merger)
			}

		case gnmiErr := <-subErrChan:
			//traverseTree(root)
			StreamObj.ForceFlush = true
			logger.Log.Infof("End of the subscription after timeout exprired - status of the end: %v", gnmiErr.Err.Error())
			StreamObj.Error = gnmiErr.Err
			time.Sleep(1 * time.Second)
			StreamObj.Result = root
			close(StreamObj.StopStreaming)
			return
		}
	}

}
