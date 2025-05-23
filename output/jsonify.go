package output

import (
	"encoding/json"
	"jtso/logger"
	"jtso/sqlite"
	"jtso/xml"
	"os"
	"strconv"
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

	// For easy search for optical mapping
	mapDesc := make(map[string]string)

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
		if strings.Contains(phy_name, "et-") || strings.Contains(phy_name, "xe-") || strings.Contains(phy_name, "ge-") || strings.Contains(phy_name, "ae") || strings.Contains(phy_name, "lt-") || strings.Contains(phy_name, "ps-") || strings.Contains(phy_name, "fti-") || strings.Contains(phy_name, "gr-") {

			_, ok := m.Meta[rd.Family][rd.RtrName][phy_name]
			if !ok {
				m.Meta[rd.Family][rd.RtrName][phy_name] = make(map[string]string)
			}
			//Default description TAG
			m.Meta[rd.Family][rd.RtrName][phy_name]["DESC"] = "Unknown"
			if strings.Contains(phy_name, "et-") || strings.Contains(phy_name, "xe-") || strings.Contains(phy_name, "ge-") {
				m.Meta[rd.Family][rd.RtrName][phy_name]["port_name"] = phy_name[3:] + " - Unknown"
				if strings.Contains(phy_name, ":") {
					m.Meta[rd.Family][rd.RtrName][phy_name]["channel"] = "yes"
				} else {
					m.Meta[rd.Family][rd.RtrName][phy_name]["channel"] = "no"
				}
			}

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

					//add to the map
					if len(phy_name) > 3 {

						if strings.Contains(phy_name, "et-") || strings.Contains(phy_name, "xe-") || strings.Contains(phy_name, "ge-") {
							m.Meta[rd.Family][rd.RtrName][phy_name]["port_name"] = phy_name[3:] + " - " + strings.ToUpper(strings.Replace(strings.Replace(phy2_desc, " ", "", -1), "-", "_", -1))
							mapDesc[phy_name[3:]] = strings.ToUpper(strings.Replace(strings.Replace(phy2_desc, " ", "", -1), "-", "_", -1))
						}
					}

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
	m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["SHORTNAME"] = strings.Trim(rtr.Shortname, "\n")
	m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["FAMILY"] = strings.Trim(family, "\n")
	if rtr.Version != "" {
		m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["VERSION"] = strings.Trim(rtr.Version, "\n")
	}

	// Add ISIS overview
	for _, isis := range rd.IsisInfo.Overview {
		ipv4Label := ""
		ipv6Label := ""

		label := strings.Trim(isis.Spring.SRGB.FirstLabel, "\n")
		if label == "" {
			continue
		}
		// Parse the first label
		numLabel, err := strconv.Atoi(label)
		if err != nil {
			logger.Log.Errorf("Unable to parse the first label from ISIS for %s: %v", rd.RtrName, err)
			continue
		}
		// Get index for IPv4
		ipv4Node := strings.Trim(isis.Spring.NodeSeg.IPv4, "\n")
		if ipv4Node != "" {
			ipv4Num, err := strconv.Atoi(ipv4Node)
			if err == nil {
				ipv4Label = strconv.Itoa(numLabel + ipv4Num)
			} else {
				logger.Log.Errorf("Unable to parse the ipv4Node from ISIS for %s: %v", rd.RtrName, err)
			}
		}
		// Get index for IPv6
		ipv6Node := strings.Trim(isis.Spring.NodeSeg.IPv6, "\n")
		if ipv6Node != "" {
			ipv6Num, err := strconv.Atoi(ipv6Node)
			if err == nil {
				ipv6Label = strconv.Itoa(numLabel + ipv6Num)
			} else {
				logger.Log.Errorf("Unable to parse the ipv6Node from ISIS for %s: %v", rd.RtrName, err)
			}
		}
		// Add the labels to the map
		if ipv4Label != "" {
			m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["MPLS_V4_SID"] = ipv4Node
			m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["MPLS_V4_LABEL"] = ipv4Label
		}
		if ipv6Label != "" {
			m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["MPLS_V6_SID"] = ipv6Node
			m.Meta[rd.Family][rd.RtrName]["LEVEL1TAGS"]["MPLS_V6_LABEL"] = ipv6Label
		}
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
									portSlot := strings.Replace(sssmSlot, "Xcvr", "", 1)
									opticDesc := sssm.Desc
									key1 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0"
									key2 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0:OCH"

									_, ok := m.Meta[rd.Family][rd.RtrName][key1]
									if !ok {
										m.Meta[rd.Family][rd.RtrName][key1] = make(map[string]string)
									}
									_, ok = m.Meta[rd.Family][rd.RtrName][key2]
									if !ok {
										m.Meta[rd.Family][rd.RtrName][key2] = make(map[string]string)
									}

									// Search if a cage match a port
									cageSlot := fpcSlot + "/" + picSlot + "/" + portSlot
									found := ""

									for _, phy := range rd.IfList.Physicals {
										phy_name := strings.Trim(phy.Name, "\n")
										// Keep only WAN ports
										if strings.Contains(phy_name, "et-") || strings.Contains(phy_name, "xe-") || strings.Contains(phy_name, "ge-") {
											if strings.Contains(phy_name, cageSlot) {
												found = phy_name
												break
											}
										}
									}
									if found != "" {
										portDesc := "Unknown"
										if len(found) > 3 {
											cageDesc, ok := mapDesc[found[3:]]
											if ok {
												portDesc = cageDesc
											}
										}

										// Add optic desc Tag
										m.Meta[rd.Family][rd.RtrName][key1]["optic_desc"] = opticDesc
										m.Meta[rd.Family][rd.RtrName][key2]["optic_desc"] = opticDesc

										// is channelized port ?
										if strings.Contains(found, ":") {
											// Channelized port
											m.Meta[rd.Family][rd.RtrName][key1]["channel"] = "yes"
											m.Meta[rd.Family][rd.RtrName][key2]["channel"] = "yes"
											m.Meta[rd.Family][rd.RtrName][key1]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
											m.Meta[rd.Family][rd.RtrName][key2]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
											m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"] = ""
											m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"] = ""
											for channel := 0; channel < 4; channel++ {
												key3 := key2 + strconv.Itoa(channel)
												_, ok = m.Meta[rd.Family][rd.RtrName][key3]
												if !ok {
													m.Meta[rd.Family][rd.RtrName][key3] = make(map[string]string)
												}
												portDesc = "Unknown"
												if len(found) > 3 {
													cageDesc, ok := mapDesc[fpcSlot+"/"+picSlot+"/"+portSlot+":"+strconv.Itoa(channel)]
													if ok {
														portDesc = cageDesc
													}
												}
												m.Meta[rd.Family][rd.RtrName][key3]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + ":" + strconv.Itoa(channel) + " - " + portDesc
												m.Meta[rd.Family][rd.RtrName][key3]["channel"] = "yes"
												m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"] += fpcSlot + "/" + picSlot + "/" + portSlot + ":" + strconv.Itoa(channel) + " ; "
												m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"] += fpcSlot + "/" + picSlot + "/" + portSlot + ":" + strconv.Itoa(channel) + " ; "
											}
											m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"] = m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"][:len(m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"])-3]
											m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"] = m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"][:len(m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"])-3]
										} else {
											// non channelized port
											m.Meta[rd.Family][rd.RtrName][key1]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
											m.Meta[rd.Family][rd.RtrName][key1]["channel"] = "no"

											m.Meta[rd.Family][rd.RtrName][key2]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
											m.Meta[rd.Family][rd.RtrName][key2]["channel"] = "no"

										}
									}
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
							opticDesc := ssm.Desc
							key1 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0"
							key2 := "FPC" + fpcSlot + ":PIC" + picSlot + ":PORT" + portSlot + ":Xcvr0:OCH"

							_, ok := m.Meta[rd.Family][rd.RtrName][key1]
							if !ok {
								m.Meta[rd.Family][rd.RtrName][key1] = make(map[string]string)
							}
							_, ok = m.Meta[rd.Family][rd.RtrName][key2]
							if !ok {
								m.Meta[rd.Family][rd.RtrName][key2] = make(map[string]string)
							}

							// Search if a cage match a port
							cageSlot := fpcSlot + "/" + picSlot + "/" + portSlot
							found := ""
							for _, phy := range rd.IfList.Physicals {
								phy_name := strings.Trim(phy.Name, "\n")
								// Keep only WAN ports
								if strings.Contains(phy_name, "et-") || strings.Contains(phy_name, "xe-") || strings.Contains(phy_name, "ge-") {
									if strings.Contains(phy_name, cageSlot) {
										found = phy_name
										break
									}
								}
							}
							if found != "" {
								portDesc := "Unknown"
								if len(found) > 3 {
									cageDesc, ok := mapDesc[found[3:]]
									if ok {
										portDesc = cageDesc
									}
								}

								// Add optic desc Tag
								m.Meta[rd.Family][rd.RtrName][key1]["optic_desc"] = opticDesc
								m.Meta[rd.Family][rd.RtrName][key2]["optic_desc"] = opticDesc

								// is channelized port ?
								if strings.Contains(found, ":") {
									// Channelized port
									m.Meta[rd.Family][rd.RtrName][key1]["channel"] = "yes"
									m.Meta[rd.Family][rd.RtrName][key2]["channel"] = "yes"
									m.Meta[rd.Family][rd.RtrName][key1]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
									m.Meta[rd.Family][rd.RtrName][key2]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
									m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"] = ""
									m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"] = ""
									for channel := 0; channel < 4; channel++ {
										key3 := key2 + strconv.Itoa(channel)
										_, ok = m.Meta[rd.Family][rd.RtrName][key3]
										if !ok {
											m.Meta[rd.Family][rd.RtrName][key3] = make(map[string]string)
										}
										portDesc = "Unknown"
										if len(found) > 3 {
											cageDesc, ok := mapDesc[fpcSlot+"/"+picSlot+"/"+portSlot+":"+strconv.Itoa(channel)]
											if ok {
												portDesc = cageDesc
											}
										}
										m.Meta[rd.Family][rd.RtrName][key3]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + ":" + strconv.Itoa(channel) + " - " + portDesc
										m.Meta[rd.Family][rd.RtrName][key3]["channel"] = "yes"
										m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"] += fpcSlot + "/" + picSlot + "/" + portSlot + ":" + strconv.Itoa(channel) + " ; "
										m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"] += fpcSlot + "/" + picSlot + "/" + portSlot + ":" + strconv.Itoa(channel) + " ; "
									}
									m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"] = m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"][:len(m.Meta[rd.Family][rd.RtrName][key1]["sub_ports"])-3]
									m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"] = m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"][:len(m.Meta[rd.Family][rd.RtrName][key2]["sub_ports"])-3]
								} else {
									// non channelized port
									m.Meta[rd.Family][rd.RtrName][key1]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
									m.Meta[rd.Family][rd.RtrName][key1]["channel"] = "no"

									m.Meta[rd.Family][rd.RtrName][key2]["port_name"] = fpcSlot + "/" + picSlot + "/" + portSlot + " - " + portDesc
									m.Meta[rd.Family][rd.RtrName][key2]["channel"] = "no"

								}
							}

						}
					}
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
