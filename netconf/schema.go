package netconf

import (
	"encoding/xml"
	"fmt"
	"jtso/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
	"golang.org/x/crypto/ssh"
)

const YANG_PATH = "/var/yang/"

// schemaList represents the NETCONF monitoring schemas response
type schemaList struct {
	XMLName xml.Name `xml:"data"`
	State   struct {
		Schemas struct {
			Schema []schemaEntry `xml:"schema"`
		} `xml:"schemas"`
	} `xml:"netconf-state"`
}

type schemaEntry struct {
	Identifier string `xml:"identifier"`
}

// getSchemaData represents the get-schema RPC reply
type getSchemaData struct {
	XMLName xml.Name `xml:"data"`
	Data    string   `xml:",chardata"`
}

// DownloadYangSchemas connects to a router via NETCONF, retrieves the list of
// available YANG schemas, filters out any whose name contains a substring from
// excludeFiles, and downloads the remaining schemas to outputDir.
// Returns the number of successfully downloaded schemas and any error.
func DownloadYangSchemas(router string, port int, username string, password string, excludeFiles []string, outputDir string) (int, error) {

	logger.Log.Infof("[%s] Start downloading YANG schemas", router)

	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	session, err := netconf.DialSSH(fmt.Sprintf("%s:%d", router, port), sshConfig)
	if err != nil {
		logger.Log.Errorf("[%s] Unable to open Netconf session for schema download: %v", router, err)
		return 0, err
	}
	defer session.Close()

	capabilities := netconf.DefaultCapabilities
	err = session.SendHello(&message.Hello{Capabilities: capabilities})
	if err != nil {
		logger.Log.Errorf("[%s] Error while sending Hello: %v", router, err)
		return 0, err
	}

	// Step 1: Get the list of available schemas
	getFilter := `<get><filter type="subtree"><netconf-state xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring"><schemas/></netconf-state></filter></get>`
	rpc := message.NewRPC(getFilter)
	reply, err := session.SyncRPC(rpc, 60)
	if err != nil || reply == nil {
		logger.Log.Errorf("[%s] Unable to retrieve schema list: %v", router, err)
		return 0, fmt.Errorf("unable to retrieve schema list: %w", err)
	}
	if strings.Contains(reply.Data, "<rpc-error>") {
		logger.Log.Errorf("[%s] RPC error while retrieving schema list", router)
		return 0, fmt.Errorf("RPC error in schema list response")
	}

	// Parse the schema list
	var schemas schemaList
	if err := xml.Unmarshal([]byte(reply.Data), &schemas); err != nil {
		logger.Log.Errorf("[%s] Unable to parse schema list: %v", router, err)
		return 0, fmt.Errorf("unable to parse schema list: %w", err)
	}

	allSchemas := schemas.State.Schemas.Schema
	if len(allSchemas) == 0 {
		logger.Log.Warnf("[%s] No YANG schemas found on device", router)
		return 0, nil
	}
	logger.Log.Infof("[%s] Found %d schemas on device", router, len(allSchemas))

	// Create output directory
	if err := os.MkdirAll(YANG_PATH+outputDir, 0755); err != nil {
		return 0, fmt.Errorf("unable to create output directory %s: %w", YANG_PATH+outputDir, err)
	}

	// Step 2: Download each schema (filtering out excluded ones)
	downloaded := 0
	for _, s := range allSchemas {
		if shouldExclude(s.Identifier, excludeFiles) {
			logger.Log.Debugf("[%s] Skipping excluded schema: %s", router, s.Identifier)
			continue
		}

		err := downloadSingleSchema(session, s.Identifier, YANG_PATH+outputDir)
		if err != nil {
			logger.Log.Warnf("[%s] Failed to download schema %s: %v", router, s.Identifier, err)
			continue
		}
		downloaded++
	}

	logger.Log.Infof("[%s] Successfully downloaded %d YANG schemas to %s", router, downloaded, YANG_PATH+outputDir)
	return downloaded, nil
}

// shouldExclude checks if a schema name contains any of the exclude substrings
func shouldExclude(name string, excludeFiles []string) bool {
	for _, excl := range excludeFiles {
		if strings.Contains(name, excl) {
			return true
		}
	}
	return false
}

// downloadSingleSchema downloads a single YANG schema from the device
func downloadSingleSchema(session *netconf.Session, module string, outputDir string) error {
	rpcData := fmt.Sprintf(`<get-schema xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring"><identifier>%s</identifier><format>yang</format></get-schema>`, module)
	rpc := message.NewRPC(rpcData)
	reply, err := session.SyncRPC(rpc, 60)
	if err != nil || reply == nil {
		return fmt.Errorf("RPC failed: %w", err)
	}
	if strings.Contains(reply.Data, "<rpc-error>") {
		return fmt.Errorf("RPC error in get-schema response")
	}

	// Parse the schema content from the reply
	var data getSchemaData
	if err := xml.Unmarshal([]byte(reply.Data), &data); err != nil {
		return fmt.Errorf("unable to parse get-schema reply: %w", err)
	}

	content := strings.TrimSpace(data.Data)
	if content == "" {
		return fmt.Errorf("empty schema content")
	}

	// Write to file
	filePath := filepath.Join(outputDir, module+".yang")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("unable to write file %s: %w", filePath, err)
	}

	return nil
}
