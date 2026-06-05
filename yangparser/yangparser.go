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

// Exporter holds pre-computed data for a YANG directory so that multiple
// Export calls don't re-scan the filesystem each time.
type Exporter struct {
	yangDir      string
	expandedPath []string
	// augFiles maps moduleName -> list of augmentation file paths
	augFiles map[string][]string
	// devFiles maps moduleName -> list of deviation file paths
	devFiles map[string][]string
	// commonDevFile is the path to jnx-openconfig-dev.yang (empty if not present)
	commonDevFile string
}

// NewExporter creates an Exporter for the given YANG directory.
// It pre-scans the directory for augmentation and deviation files.
func NewExporter(yangDir string) (*Exporter, error) {
	expanded, err := yang.PathsWithModules(yangDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand YANG dir %s: %w", yangDir, err)
	}

	ex := &Exporter{
		yangDir:      yangDir,
		expandedPath: expanded,
		augFiles:     make(map[string][]string),
		devFiles:     make(map[string][]string),
	}

	// Pre-scan augmentation files: jnx-aug-*.yang
	augPattern := filepath.Join(yangDir, "jnx-aug-*.yang")
	augMatches, _ := filepath.Glob(augPattern)
	for _, f := range augMatches {
		base := filepath.Base(f)
		// Extract module name from jnx-aug-<moduleName>*.yang
		name := strings.TrimPrefix(base, "jnx-aug-")
		name = strings.TrimSuffix(name, ".yang")
		// Remove any trailing version suffix (e.g., "-1.0")
		// The key is the openconfig module being augmented
		// We'll match by prefix at lookup time
		ex.augFiles[base] = append(ex.augFiles[base], f)
	}

	// Pre-scan deviation files: jnx-*-dev*.yang (excluding jnx-openconfig-dev.yang)
	devPattern := filepath.Join(yangDir, "jnx-*-dev*.yang")
	devMatches, _ := filepath.Glob(devPattern)
	for _, f := range devMatches {
		base := filepath.Base(f)
		if base == "jnx-openconfig-dev.yang" {
			continue
		}
		ex.devFiles[base] = append(ex.devFiles[base], f)
	}

	// Check for common deviation file
	commonDev := filepath.Join(yangDir, "jnx-openconfig-dev.yang")
	if _, err := os.Stat(commonDev); err == nil {
		ex.commonDevFile = commonDev
	}

	return ex, nil
}

// Export parses a single YANG module and writes all state (read-only) leaf
// paths as JSON into yangDir with the same base name but .json extension.
// If withType is true, the type name is included in XType; otherwise XType is "not-checked".
func (ex *Exporter) Export(yangFile string, withType bool) error {
	ms := yang.NewModules()
	ms.AddPath(ex.expandedPath...)

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

	// Load augmentation and deviation files only for openconfig modules
	if strings.HasPrefix(moduleName, "openconfig") {
		ex.loadAugmentAndDeviation(ms, moduleName)
	}

	// Process the modules — ignore errors from deviations targeting
	// nodes not present in this module (e.g. jnx-openconfig-dev.yang
	// contains deviations for many modules).
	ms.Process()

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
	outputPath := filepath.Join(ex.yangDir, jsonName)

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", outputPath, err)
	}

	return nil
}

// Export is a convenience function that creates a one-shot Exporter.
// For batch processing, use NewExporter + ex.Export in a loop instead.
func Export(yangDir string, yangFile string, withType bool) error {
	ex, err := NewExporter(yangDir)
	if err != nil {
		return err
	}
	return ex.Export(yangFile, withType)
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

// loadAugmentAndDeviation loads pre-scanned augmentation and deviation files
// that target the given module into ms.
func (ex *Exporter) loadAugmentAndDeviation(ms *yang.Modules, moduleName string) {
	// Load augmentation files: jnx-aug-<moduleName>*.yang
	for base, files := range ex.augFiles {
		if strings.HasPrefix(base, "jnx-aug-"+moduleName) {
			for _, f := range files {
				_ = ms.Read(f)
			}
		}
	}

	// Load deviation files: jnx-<moduleName>-dev*.yang
	for base, files := range ex.devFiles {
		if strings.HasPrefix(base, "jnx-"+moduleName+"-dev") {
			for _, f := range files {
				_ = ms.Read(f)
			}
		}
	}

	// Load common deviation file
	if ex.commonDevFile != "" {
		_ = ms.Read(ex.commonDevFile)
	}
}
