package yangparser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// FlatPath represents a single YANG leaf path with its metadata.
type FlatPath struct {
	ReadOnly bool   `json:"read_only"`
	Xpath    string `json:"xpath"`
	XDesc    string `json:"xdesc"`
	XType    string `json:"xtype"`
}

// Export parses the YANG module located at yangFile using the YANG directory
// provided in yangDir, and writes all state (read-only) leaf paths as JSON
// into yangDir with the same base name as yangFile but with .json extension.
// If withType is true, the type name is included in XType; otherwise XType is "not-checked".
func Export(yangDir string, yangFile string, withType bool) error {
	// Read the module
	ms := yang.NewModules()

	// Add YANG directory to the search path
	expanded, err := yang.PathsWithModules(yangDir)
	if err != nil {
		return fmt.Errorf("failed to expand YANG dir %s: %w", yangDir, err)
	}
	ms.AddPath(expanded...)

	if err := ms.Read(yangFile); err != nil {
		return fmt.Errorf("failed to read YANG module %s: %w", yangFile, err)
	}
	if len(ms.Modules) == 0 {
		// This is likely a submodule (belongs-to another module), skip silently
		return nil
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

	// Load augmentation and deviation files that target this module
	loadAugmentAndDeviation(ms, yangDir, moduleName)

	// Process the modules
	errs := ms.Process()
	if len(errs) > 0 {
		return fmt.Errorf("YANG processing errors: %v", errs)
	}

	entry := yang.ToEntry(ms.Modules[moduleName])

	// Traverse the tree and collect state leaf paths
	var results []FlatPath
	traverseEntry(entry, "", yang.TSUnset, withType, &results)

	// Marshal results to JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", err)
	}

	// Write JSON file with same base name as yangFile but .json extension
	baseName := filepath.Base(yangFile)
	jsonName := strings.TrimSuffix(baseName, ".yang") + ".json"
	outputPath := filepath.Join(yangDir, jsonName)

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", outputPath, err)
	}

	return nil
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

// loadAugmentAndDeviation scans yangDir for jnx-aug-* and jnx-*-dev* files
// that augment or deviate the given module, and loads them into ms.
func loadAugmentAndDeviation(ms *yang.Modules, yangDir string, moduleName string) {
	// Look for augmentation files: jnx-aug-<moduleName>.yang
	augPattern := filepath.Join(yangDir, "jnx-aug-"+moduleName+"*.yang")
	augFiles, _ := filepath.Glob(augPattern)
	for _, f := range augFiles {
		_ = ms.Read(f)
	}

	// Look for deviation files: jnx-<moduleName>-dev*.yang
	devPattern := filepath.Join(yangDir, "jnx-"+moduleName+"-dev*.yang")
	devFiles, _ := filepath.Glob(devPattern)
	for _, f := range devFiles {
		_ = ms.Read(f)
	}

	// Also check for the common deviation file: jnx-openconfig-dev.yang
	commonDev := filepath.Join(yangDir, "jnx-openconfig-dev.yang")
	if _, err := os.Stat(commonDev); err == nil {
		_ = ms.Read(commonDev)
	}
}
