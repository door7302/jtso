package output

import (
	"encoding/json"
	"jtso/logger"
	"jtso/xml"
	"os"
	"strings"
	"sync"
)

type Metadata struct {
	Mu *sync.Mutex
	// 2 levels enrichment map
	// Level 1 Router as a key
	// Level 2 - Wellknown key LEVEL1TAGS or any other L2 key (interface, MPC...)
	Meta map[string]map[string]map[string]map[string]string
}

var MyMeta *Metadata

// Initialize the new meta map
func init() {
	// init the metadata
	MyMeta = &Metadata{
		Mu:   new(sync.Mutex),
		Meta: make(map[string]map[string]map[string]map[string]string),
	}
}

// Clear the Meta map
func (m *Metadata) Clear() {
	m.Mu.Lock()
	m.Meta = make(map[string]map[string]map[string]map[string]string)
	m.Mu.Unlock()
}

// Clear the Meta map for a given router
func (m *Metadata) ClearRtr(p string, r string) {
	m.Mu.Lock()
	_, ok := m.Meta[p]
	if !ok {
		m.Mu.Unlock()
		return
	}
	_, ok = m.Meta[p][r]
	if !ok {
		m.Mu.Unlock()
		return
	}
	m.Meta[p][r] = make(map[string]map[string]string)
	m.Mu.Unlock()
}

// Update the map for a given router
func (m *Metadata) UpdateMeta(rd *xml.RawData) error {
	m.Mu.Lock()

	// init Map
	_, ok := m.Meta[rd.Family]
	if !ok {
		m.Meta[rd.Family] = make(map[string]map[string]map[string]string)
	}
	_, ok = m.Meta[rd.Family][rd.RtrName]
	if !ok {
		m.Meta[rd.Family][rd.RtrName] = map[string]map[string]string{}
	}

	// ADD physical / logical interface description
	for _, phy := range rd.IfInfo.Physicals {
		phy_name := strings.Trim(phy.Name, "\n")
		phy_desc := strings.Trim(phy.Desc, "\n")
		_, ok := m.Meta[rd.Family][rd.RtrName][phy_name]
		if !ok {
			m.Meta[rd.Family][rd.RtrName][phy_name] = make(map[string]string)
		}
		m.Meta[rd.Family][rd.RtrName][phy_name]["DESC"] = strings.ToUpper(strings.Replace(strings.Replace(phy_desc, " ", "", -1), "-", "_", -1))
		// Add also the parent LAG name if physical interface is a child link.
		val, ok := rd.LacpDigest.LacpMap[phy_name]
		if ok {
			m.Meta[rd.Family][rd.RtrName][phy_name]["LAG"] = val
		}
	}
	for _, lgl := range rd.IfInfo.Logicals {
		lgl_name := strings.Trim(lgl.Name, "\n")
		lgl_desc := strings.Trim(lgl.Desc, "\n")
		_, ok := m.Meta[rd.Family][rd.RtrName][lgl_name]
		if !ok {
			m.Meta[rd.Family][rd.RtrName][lgl_name] = make(map[string]string)
		}
		m.Meta[rd.Family][rd.RtrName][lgl_name]["DESC"] = strings.ToUpper(strings.Replace(strings.Replace(lgl_desc, " ", "", -1), "-", "_", -1))
	}
	// add HW info
	// Chassis model
	_, ok = m.Meta[rd.Family][rd.RtrName]["LEVEL1TAG"]
	if !ok {
		m.Meta[rd.Family][rd.RtrName]["LEVEL1TAG"] = make(map[string]string)
	}
	m.Meta[rd.Family][rd.RtrName]["LEVEL1TAG"]["MODEL"] = strings.Trim(rd.HwInfo.Chassis.Desc, "\n")
	// For each LC add a TAG
	for _, mod := range rd.HwInfo.Chassis.Modules {
		slot := strings.Trim(strings.Replace(mod.Name, " ", "", 1), "\n")
		if strings.Contains(slot, "FPC") {
			_, ok := m.Meta[rd.Family][rd.RtrName][slot]
			if !ok {
				m.Meta[rd.Family][rd.RtrName][slot] = make(map[string]string)
			}
			m.Meta[rd.Family][rd.RtrName][slot]["HW_TYPE"] = strings.Trim(mod.Desc, "\n")
		}
	}
	m.Mu.Unlock()
	return nil
}

// Create the Json file
func (m *Metadata) MarshallMeta(f string) error {
	m.Mu.Lock()

	for k, v := range m.Meta {

		json, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			m.Mu.Unlock()
			return err
		}
		logger.Log.Info("Target is:" + f + "metadata_" + k + ".json")
		err = os.WriteFile(f+"metadata_"+k+".json", json, 0666)
		if err != nil {
			logger.Log.Info("ISSUE")
			m.Mu.Unlock()
			return err
		}
		logger.Log.Infof("Metadata file for %s Family has been generated", k)
	}
	m.Mu.Unlock()
	return nil
}
