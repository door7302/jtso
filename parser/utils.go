package parser

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

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

	re1 := regexp.MustCompile("(\\d+)")
	re2 := regexp.MustCompile("(.*)\\[(.*)=(.*)\\]")

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
