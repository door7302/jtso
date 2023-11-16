package output

import (
	"encoding/json"
	"jtso/logger"
	"jtso/sqlite"
	"jtso/xml"
	"os"
	"strings"
	"sync"
)

type Metadata struct {
	Mu *sync.Mutex
	// 2 levels enrichment map
	// Level 1 Router as a key
	// Level 2 - Wellknown key LEVEL1TAGSS or any other L2 key (interface, MPC...)
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

	for _, phy := range rd.IfList.Physicals {
		phy_name := strings.Trim(phy.Name, "\n")
		// Keep only WAN ports
		if strings.Contains(phy_name, "et-") || strings.Contains(phy_name, "xe-") || strings.Contains(phy_name, "ge-") {
			_, ok := m.Meta[rd.Family][rd.RtrName][phy_name]
			if !ok {
				m.Meta[rd.Family][rd.RtrName][phy_name] = make(map[string]string)
			}
			//Default description TAG
			m.Meta[rd.Family][rd.RtrName][phy_name]["DESC"] = "Unknown"
			m.Meta[rd.Family][rd.RtrName][phy_name]["LINKNAME"] = phy_name + " - " + "Unknown"

			// Add also the parent LAG name if physical interface is a child link.
			val, ok := rd.LacpDigest.LacpMap[phy_name]
			if ok {
				m.Meta[rd.Family][rd.RtrName][phy_name]["LAG"] = val
			}

			// check if PHY port has a description
			// ADD physical description if present
			for _, phy2 := range rd.IfDesc.Physicals {
				phy2_name := strings.Trim(phy2.Name, "\n")
				phy2_desc := strings.Trim(phy2.Desc, "\n")

				if phy2_name == phy_name && phy2_desc != "" {
					m.Meta[rd.Family][rd.RtrName][phy_name]["LINKNAME"] = phy2_name + " - " + strings.ToUpper(strings.Replace(strings.Replace(phy2_desc, " ", "", -1), "-", "_", -1))
					m.Meta[rd.Family][rd.RtrName][phy_name]["DESC"] = strings.ToUpper(strings.Replace(strings.Replace(phy2_desc, " ", "", -1), "-", "_", -1))
				}
			}
		}
	}

	// ADD logical description
	for _, lgl := range rd.IfDesc.Logicals {
		lgl_name := strings.Trim(lgl.Name, "\n")
		lgl_desc := strings.Trim(lgl.Desc, "\n")
		_, ok := m.Meta[rd.Family][rd.RtrName][lgl_name]
		if !ok {
			m.Meta[rd.Family][rd.RtrName][lgl_name] = make(map[string]string)
		}
		m.Meta[rd.Family][rd.RtrName][lgl_name]["DESC"] = strings.ToUpper(strings.Replace(strings.Replace(lgl_desc, " ", "", -1), "-", "_", -1))
	}
	// add HW info
	// Chassis model + Version

	// Find out the router entry to extract version already collected by the get Facts
	var rtr *sqlite.RtrEntry
	rtr = new(sqlite.RtrEntry)
	family := ""
	for _, r := range sqlite.RtrList {
		if r.Hostname == rd.RtrName {
			rtr = r
			family = r.Family
		}
	}
	_, ok = m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]
	if !ok {
		m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"] = make(map[string]string)
	}
	m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["MODEL"] = strings.Trim(rd.HwInfo.Chassis.Desc, "\n")
	if rtr.Version != "" {
		m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["VERSION"] = strings.Trim(rtr.Version, "\n")
	}

	// For each LC add a TAG
	for _, mod := range rd.HwInfo.Chassis.Modules {
		mSlot := strings.Trim(strings.Replace(mod.Name, " ", "", 1), "\n")

		if strings.Contains(mSlot, "FPC") {
			fpcSlot := strings.Replace(mSlot, "FPC", "", 1)
			_, ok := m.Meta[rd.Family][rd.RtrName][mSlot]
			if !ok {
				m.Meta[rd.Family][rd.RtrName][mSlot] = make(map[string]string)
			}
			m.Meta[rd.Family][rd.RtrName][mSlot]["HW_TYPE"] = strings.Trim(mod.Desc, "\n")
			for _, sm := range mod.SubMods {
				smSlot := strings.Trim(strings.Replace(sm.Name, " ", "", 1), "\n")
				if strings.Contains(smSlot, "MIC") {
					for _, ssm := range sm.SubSubMods {
						ssmSlot := strings.Trim(strings.Replace(ssm.Name, " ", "", 1), "\n")
						if strings.Contains(ssmSlot, "PIC") {
							picSlot := strings.Replace(ssmSlot, "PIC", "", 1)
							for _, sssm := range ssm.SubSubSubMods {
								sssmSlot := strings.Trim(strings.Replace(sssm.Name, " ", "", 1), "\n")
								if strings.Contains(sssmSlot, "Xcvr") {
									portSlot := strings.Replace(ssmSlot, "Xcvr", "", 1)
									prfx := "et-"
									key1 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0"
									key2 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0:OCH"
									if family == "mx" || family == "vmx" {
										if strings.Contains(sssm.Desc, "1G") {
											prfx = "ge-"
										} else if strings.Contains(sssm.Desc, "1G") {
											prfx = "xe-"
										}
									}
									_, ok := m.Meta[rd.Family][rd.RtrName][key1]
									if !ok {
										m.Meta[rd.Family][rd.RtrName][key1] = make(map[string]string)
									}
									_, ok = m.Meta[rd.Family][rd.RtrName][key2]
									if !ok {
										m.Meta[rd.Family][rd.RtrName][key2] = make(map[string]string)
									}
									m.Meta[rd.Family][rd.RtrName][key1]["if_name"] = prfx + fpcSlot + "/" + picSlot + "/" + portSlot
									m.Meta[rd.Family][rd.RtrName][key2]["if_name"] = prfx + fpcSlot + "/" + picSlot + "/" + portSlot
								}
							}
						}

					}
				}
				if strings.Contains(smSlot, "PIC") {
					picSlot := strings.Replace(smSlot, "PIC", "", 1)
					for _, ssm := range sm.SubSubMods {
						ssmSlot := strings.Trim(strings.Replace(ssm.Name, " ", "", 1), "\n")
						if strings.Contains(ssmSlot, "Xcvr") {
							portSlot := strings.Replace(ssmSlot, "Xcvr", "", 1)
							prfx := "et-"
							key1 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0"
							key2 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0:OCH"
							if family == "mx" || family == "vmx" {
								if strings.Contains(ssm.Desc, "1G") {
									prfx = "ge-"
								} else if strings.Contains(ssm.Desc, "1G") {
									prfx = "xe-"
								}
							}
							_, ok := m.Meta[rd.Family][rd.RtrName][key1]
							if !ok {
								m.Meta[rd.Family][rd.RtrName][key1] = make(map[string]string)
							}
							_, ok = m.Meta[rd.Family][rd.RtrName][key2]
							if !ok {
								m.Meta[rd.Family][rd.RtrName][key2] = make(map[string]string)
							}
							m.Meta[rd.Family][rd.RtrName][key1]["if_name"] = prfx + fpcSlot + "/" + picSlot + "/" + portSlot
							m.Meta[rd.Family][rd.RtrName][key2]["if_name"] = prfx + fpcSlot + "/" + picSlot + "/" + portSlot

						}
					}
				}
			}
		}
	}

	// Derive a tag for HW optic to assign component_name to if_name
	for _, m := range rd.HwInfo.Chassis.Modules {

		for _, sm := range m.SubMods {
			logger.Log.Debugf(" │  ├─ Sub-Module: %s - %s", strings.Trim(sm.Name, "\n"), strings.Trim(sm.Desc, "\n"))
			for _, ssm := range sm.SubSubMods {
				logger.Log.Debugf(" │  │  ├─ Sub-Sub-Module: %s - %s", strings.Trim(ssm.Name, "\n"), strings.Trim(ssm.Desc, "\n"))
				for _, sssm := range ssm.SubSubSubMods {
					logger.Log.Debugf(" │  │  │  ├─ Sub-Sub-Sub-Module: %s - %s", strings.Trim(sssm.Name, "\n"), strings.Trim(sssm.Desc, "\n"))
				}
			}
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
		if f[len(f)-1:] != "/" {
			f += "/"
		}

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
