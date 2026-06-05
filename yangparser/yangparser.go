package yangparser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// FlatPath represents a single YANG leaf path with its metadata.
type FlatPath struct {
	ReadOnly bool
	Xpath    string
	XDesc    string
	XType    string
}

// Export parses the YANG module located at yangFile using the YANG directories
// provided in yangDirs, and returns all state (read-only) leaf paths.
// If withType is true, the type name is included in XType; otherwise XType is "not-checked".
func Export(yangDirs []string, yangFile string, withType bool) ([]FlatPath, error) {
	// Read the module
	ms := yang.NewModules()

	// Add YANG directories to the search path
	for _, dir := range yangDirs {
		expanded, err := yang.PathsWithModules(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to expand YANG dir %s: %w", dir, err)
		}
		ms.AddPath(expanded...)
	}

	if err := ms.Read(yangFile); err != nil {
		return nil, fmt.Errorf("failed to read YANG module %s: %w", yangFile, err)
	}
	if len(ms.Modules) == 0 {
		return nil, fmt.Errorf("no YANG modules found for %s", yangFile)
	}

	// Get the module name (remove revision entries)
	var moduleName string
	for k, m := range ms.Modules {
		if strings.Contains(k, "@") {
			delete(ms.Modules, k)
			continue
		}
		moduleName = m.Name
	}

	// Process the modules
	errs := ms.Process()
	if len(errs) > 0 {
		return nil, fmt.Errorf("YANG processing errors: %v", errs)
	}

	entry := yang.ToEntry(ms.Modules[moduleName])

	// Traverse the tree and collect state leaf paths
	var results []FlatPath
	traverseEntry(entry, "", yang.TSUnset, withType, &results)

	return results, nil
}

// traverseEntry recursively walks the YANG entry tree, building xpath paths
// and collecting leaf nodes that are read-only (state).
func traverseEntry(e *yang.Entry, currentPath string, parentConfig yang.TriState, withType bool, results *[]FlatPath) {
	config := parentConfig

	switch e.Node.(type) {
	case *yang.Module:
		// module is root, no path contribution
	case *yang.Container:
		currentPath += fmt.Sprintf("/%s", e.Name)
		if e.Config != yang.TSUnset {
			config = e.Config
		}
	case *yang.List:
		if e.Config != yang.TSUnset {
			config = e.Config
		}
		var keyElem string
		if e.Key != "" {
			keys := strings.Split(e.Key, " ")
			for _, k := range keys {
				keyElem += fmt.Sprintf("[%s=*]", k)
			}
		}
		currentPath += fmt.Sprintf("/%s%s", e.Name, keyElem)
	case *yang.LeafList:
		if e.Config != yang.TSUnset {
			config = e.Config
		}
	case *yang.Leaf:
		if e.Config != yang.TSUnset {
			config = e.Config
		}
		leafPath := currentPath + fmt.Sprintf("/%s", e.Name)

		// Only collect state (read-only) nodes
		if config == yang.TSFalse {
			fp := FlatPath{
				ReadOnly: true,
				Xpath:    leafPath,
				XDesc:    e.Description,
			}
			if withType {
				fp.XType = resolveTypeName(e)
			} else {
				fp.XType = "not-checked"
			}
			*results = append(*results, fp)
		}
		return // leaf has no children
	}

	// Recurse into children sorted alphabetically
	childNames := make([]string, 0, len(e.Dir))
	for k := range e.Dir {
		childNames = append(childNames, k)
	}
	sort.Strings(childNames)

	for _, name := range childNames {
		traverseEntry(e.Dir[name], currentPath, config, withType, results)
	}
}

// resolveTypeName extracts the type name from a leaf entry, including
// additional info for identityref, leafref, enum, and union types.
func resolveTypeName(e *yang.Entry) string {
	leaf, ok := e.Node.(*yang.Leaf)
	if !ok || leaf.Type == nil {
		return ""
	}

	typeName := leaf.Type.Name

	// identityref
	if leaf.Type.IdentityBase != nil {
		typeName += fmt.Sprintf("->%v", leaf.Type.IdentityBase.Name)
	}

	// leafref
	if e.Type != nil && e.Type.Kind == yang.Yleafref {
		typeName += fmt.Sprintf("->%v", e.Type.Path)
	}

	// enumeration
	if e.Type != nil && e.Type.Kind == yang.Yenum {
		typeName += fmt.Sprintf("%+q", e.Type.Enum.Names())
	}

	// union
	if e.Type != nil && e.Type.Kind == yang.Yunion {
		var u []string
		for _, ut := range leaf.Type.Type {
			switch {
			case ut.IdentityBase != nil:
				u = append(u, fmt.Sprintf("identityref->%v", ut.IdentityBase.Name))
			case ut.YangType != nil && ut.YangType.Kind == yang.Yenum:
				u = append(u, fmt.Sprintf("enumeration%+q", ut.YangType.Enum.Names()))
			default:
				u = append(u, ut.Name)
			}
		}
		typeName += fmt.Sprintf("{%v}", strings.Join(u, " "))
	}

	return typeName
}
