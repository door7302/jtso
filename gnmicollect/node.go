package gnmicollect

import "strings"

type TrieNode struct {
	children map[string]*TrieNode
	count    int
}

// TreeNode represents a node in the tree structure.
type TreeNode struct {
	Data     interface{}
	Value    map[string]interface{}
	Children []*TreeNode
}

// New creates a new instance of TreeNode with the given data and value.
func NewTree(data, value interface{}) *TreeNode {
	return &TreeNode{Data: data, Value: value.(map[string]interface{}), Children: make([]*TreeNode, 0)}
}

// AddChild adds a new child to the current node.
func (n *TreeNode) AddChild(child *TreeNode) {
	n.Children = append(n.Children, child)
}

// InsertChild inserts a new child into the tree structure.
func (n *TreeNode) InsertChild(data interface{}, value map[string]interface{}) *TreeNode {
	newChild := NewTree(data, value)
	n.AddChild(newChild)
	return newChild
}

// AddValue adds a value to the current node.
func (n *TreeNode) AddValue(value map[string]interface{}) {
	n.Value = mergeMap(n.Value, value)
	//for key, value := range value {
	//		n.Value[key] = value
	//	}
}

func mergeMap(m1 map[string]interface{}, m2 map[string]interface{}) map[string]interface{} {
	result := deepCopyMap(m1)
	for k, v := range m2 {
		found := false
		if val, ok := result[k]; ok {
			if _, isDictResult := val.(map[string]interface{}); isDictResult {
				if _, isDictValue := v.(map[string]interface{}); isDictValue {
					found = true
					result[k] = mergeMap(result[k].(map[string]interface{}), v.(map[string]interface{}))
				}
			}
		}
		if !found {
			result[k] = v
		}
	}
	return result
}

func deepCopyMap(original map[string]interface{}) map[string]interface{} {
	copyMap := make(map[string]interface{})

	for key, value := range original {
		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively deep copy nested maps
			copyMap[key] = deepCopyMap(v)
		case []interface{}:
			// Recursively deep copy nested slices
			copyMap[key] = deepCopySlice(v)
		default:
			// Copy other types directly
			copyMap[key] = value
		}
	}

	return copyMap
}

func deepCopySlice(original []interface{}) []interface{} {
	copySlice := make([]interface{}, len(original))

	for i, v := range original {
		switch element := v.(type) {
		case map[string]interface{}:
			// Recursively deep copy nested maps
			copySlice[i] = deepCopyMap(element)
		case []interface{}:
			// Recursively deep copy nested slices
			copySlice[i] = deepCopySlice(element)
		default:
			// Copy other types directly
			copySlice[i] = element
		}
	}

	return copySlice
}

// FindNode finds a node in the tree structure by searching for a specific piece of data.
func (n *TreeNode) FindNode(targetData interface{}) (*TreeNode, bool) {
	if n == nil {
		return nil, false
	}

	if n.Data == targetData {
		return n, true
	}

	for _, child := range n.Children {
		if child.Data == targetData {
			return child, true
		}
	}

	return nil, false
}

func (n *TreeNode) Traverse(f func(node *TreeNode)) {
	f(n)
	for _, child := range n.Children {
		child.Traverse(f)
	}
}

/// For gnmi once only - to auto detect aliasing

func Insert(root *TrieNode, xpath string) {
	parts := strings.Split(strings.Trim(xpath, "/"), "/")
	node := root

	for _, p := range parts {
		if node.children == nil {
			node.children = make(map[string]*TrieNode)
		}
		if _, ok := node.children[p]; !ok {
			node.children[p] = &TrieNode{}
		}
		node = node.children[p]
		node.count++
	}
}

func CollectPrefixes(node *TrieNode, path []string, result *[]string) {
	for seg, child := range node.children {
		if child.count >= 2 {
			newPath := append(path, seg)

			stop := false
			for _, gc := range child.children {
				if gc.count < 2 {
					stop = true
					break
				}
			}

			if stop || len(child.children) == 0 {
				*result = append(*result, "/"+strings.Join(newPath, "/"))
			} else {
				CollectPrefixes(child, newPath, result)
			}
		}
	}
}
