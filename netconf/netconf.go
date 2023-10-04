package netconf

import (
	"fmt"
	"jtso/logger"
	"jtso/output"
	"jtso/xml"
	"strings"
	"sync"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
	"golang.org/x/crypto/ssh"
)

// The task structure
type RouterTask struct {
	Name    string
	User    string
	Pwd     string
	Family  string
	Port    int
	Timeout int
	Wg      *sync.WaitGroup
	Jsonify *output.Metadata
}

func GetFacts(r string, u string, p string, port int) (*xml.Version, error) {

	logger.Log.Infof("[%s] Get Facts for new router - open seesion on port %d for username %s", r, port, u)

	sshConfig := &ssh.ClientConfig{
		User:            u,
		Auth:            []ssh.AuthMethod{ssh.Password(p)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var replyVersion *xml.Version

	session, err := netconf.DialSSH(fmt.Sprintf("%s:%d", r, port), sshConfig)
	if err != nil {
		logger.Log.Errorf("[%s] Unable to open Netconf session: %v", r, err)
		return nil, err
	}

	defer session.Close()
	capabilities := netconf.DefaultCapabilities
	err = session.SendHello(&message.Hello{Capabilities: capabilities})
	if err != nil {
		logger.Log.Errorf("[%s] Error while sending Hello: %v", r, err)
		return nil, err
	}

	d := "<get-software-information></get-software-information>"
	rpc := message.NewRPC(d)
	reply, err := session.SyncRPC(rpc, int32(60))
	if err != nil || reply == nil || strings.Contains(reply.Data, "<rpc-error>") {
		logger.Log.Warnf("[%s] No Version information: %v", r, err)
		return nil, err

	} else {
		// Unmarshall the reply
		replyVersion, err = xml.ParseVersion(reply.Data)
		if err != nil {
			logger.Log.Warnf("[%s] Unable to parse version information: %v", r, err)
			return nil, err
		}
	}
	return replyVersion, nil
}

// The Worker function
func (r *RouterTask) Work() error {
	logger.HandlePanic()
	defer r.Wg.Done()

	logger.Log.Infof("[%s] Start collecting and updating Metadata", r.Name)

	sshConfig := &ssh.ClientConfig{
		User:            r.User,
		Auth:            []ssh.AuthMethod{ssh.Password(r.Pwd)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var rawData *xml.RawData
	rawData = new(xml.RawData)
	rawData.IfInfo = new(xml.Ifdesc)
	rawData.HwInfo = new(xml.Hw)
	rawData.LacpInfo = new(xml.Lacp)
	rawData.LacpDigest = new(xml.LacpDigest)
	rawData.RtrName = r.Name
	rawData.Family = r.Family

	var hasIf, hasHw, hasLacp bool

	session, err := netconf.DialSSH(fmt.Sprintf("%s:%d", r.Name, r.Port), sshConfig)

	if err != nil {
		logger.Log.Errorf("[%s] Unable to open Netconf session: %v", r.Name, err)
		return err
	}

	defer session.Close()

	capabilities := netconf.DefaultCapabilities
	err = session.SendHello(&message.Hello{Capabilities: capabilities})
	if err != nil {
		logger.Log.Errorf("[%s] Error while sending Hello: %v", r.Name, err)
		return err
	}

	d := "<get-interface-information><descriptions/></get-interface-information>"
	rpc := message.NewRPC(d)
	reply, err := session.SyncRPC(rpc, int32(r.Timeout))
	if err != nil || reply == nil || strings.Contains(reply.Data, "<rpc-error>") {
		logger.Log.Warnf("[%s] No interfaces description information: %v", r.Name, err)

	} else {
		// Unmarshall the reply
		rawData.IfInfo, err = xml.ParseIfdesc(reply.Data)
		if err != nil {
			logger.Log.Warnf("[%s] Unable to parse interface description: %v", r.Name, err)
		} else {
			hasIf = true
		}

	}

	d = "<get-chassis-inventory></get-chassis-inventory>"
	rpc = message.NewRPC(d)
	reply, err = session.SyncRPC(rpc, int32(r.Timeout))
	if err != nil || reply == nil || strings.Contains(reply.Data, "<rpc-error>") {
		logger.Log.Warnf("[%s] No Chassis HW information: %v", r.Name, err)
	} else {
		// Unmarshall the reply
		rawData.HwInfo, err = xml.ParseChassis(reply.Data)
		if err != nil {
			logger.Log.Warnf("[%s] Unable to parse chassis hardware: %v", r.Name, err)
		} else {
			hasHw = true
		}
	}

	d = "<get-lacp-interface-information></get-lacp-interface-information>"
	rpc = message.NewRPC(d)
	reply, err = session.SyncRPC(rpc, int32(r.Timeout))
	if err != nil || reply == nil || strings.Contains(reply.Data, "<rpc-error>") {
		logger.Log.Warnf("[%s] No LACP Interface information: %v", r.Name, err)
	} else {
		// Unmarshall the reply
		rawData.LacpInfo, rawData.LacpDigest, err = xml.ParseLacp(reply.Data)
		if err != nil {
			logger.Log.Warnf("[%s] Unable to parse LACP Interface: %v", r.Name, err)
		} else {
			hasLacp = true
		}
	}

	// Display detail only if verbose set
	if logger.Verbose {
		logger.Log.Debug("")
		logger.Log.Debug("--------------------------------------------------------------------")
		logger.Log.Debugf("ROUTER %s:", r.Name)
		if hasIf {
			logger.Log.Debug("")
			logger.Log.Debug("----- Interface Descriptions -----")
			logger.Log.Debug("")
			logger.Log.Debug(" Physicals Intf:")
			for _, v := range rawData.IfInfo.Physicals {
				logger.Log.Debugf(" ├─ %s : %s", strings.Trim(v.Name, "\n"), strings.Trim(v.Desc, "\n"))
			}
			logger.Log.Debug("")
			logger.Log.Debug(" Logicals Intf:")
			for _, v := range rawData.IfInfo.Logicals {
				logger.Log.Debugf(" ├─ %s : %s", strings.Trim(v.Name, "\n"), strings.Trim(v.Desc, "\n"))
			}
			logger.Log.Debug("--------------------------------------------------------------------")
		}
		if hasHw {
			logger.Log.Debug("")
			logger.Log.Debug("-------- Chassis Hardware --------")
			logger.Log.Debug("")
			logger.Log.Debugf(" Chassis model: %s", strings.Trim(rawData.HwInfo.Chassis.Desc, "\n"))
			for _, m := range rawData.HwInfo.Chassis.Modules {
				logger.Log.Debugf(" ├─ Module: %s - %s", strings.Trim(m.Name, "\n"), strings.Trim(m.Desc, "\n"))
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
			logger.Log.Debug("--------------------------------------------------------------------")
		}
		if hasLacp {
			logger.Log.Debug("")
			logger.Log.Debug("-------- LACP Information -------")
			logger.Log.Debug("")
			for _, l := range rawData.LacpInfo.LacpInt {
				logger.Log.Debugf(" ├─ LAG : %s", strings.Trim(l.LacpHead.LagName, "\n"))
				for _, c := range l.LacpProto {
					logger.Log.Debugf(" │  ├─ Member link %s", strings.Trim(c.Name, "\n"))
				}
			}

			logger.Log.Debug("--------------------------------------------------------------------")
		}
		logger.Log.Debug("")

	}
	// end debug

	// update the Metadata struct for the current router
	err = r.Jsonify.UpdateMeta(rawData)
	if err != nil {
		logger.Log.Errorf("[%s] Unable to update the MetaData structure: %v", r.Name, err)
		return err
	}

	logger.Log.Infof("[%s] End of collecting and updating Metadata", r.Name)
	return nil
}
