package association

import (
	"bufio"
	"errors"
	"io"
	"jtso/config"
	"jtso/container"
	"jtso/kapacitor"
	"jtso/logger"
	"jtso/sqlite"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

const PATH_MX string = "/var/shared/telegraf/mx/telegraf.d/"
const PATH_PTX string = "/var/shared//telegraf/ptx/telegraf.d/"
const PATH_ACX string = "/var/shared//telegraf/acx/telegraf.d/"
const PATH_EX string = "/var/shared//telegraf/ex/telegraf.d/"
const PATH_QFX string = "/var/shared//telegraf/qfx/telegraf.d/"
const PATH_SRX string = "/var/shared//telegraf/srx/telegraf.d/"

const PATH_CRPD string = "/var/shared//telegraf/crpd/telegraf.d/"
const PATH_CPTX string = "/var/shared//telegraf/cptx/telegraf.d/"

const PATH_VMX string = "/var/shared//telegraf/vmx/telegraf.d/"
const PATH_VSRX string = "/var/shared/telegraf/vsrx/telegraf.d/"
const PATH_VJUNOS string = "/var/shared//telegraf/vjunos/telegraf.d/"
const PATH_VSWITCH string = "/var/shared//telegraf/vswitch/telegraf.d/"
const PATH_VEVO string = "/var/shared//telegraf/vevo/telegraf.d/"

const TELEGRAF_ROOT_PATH string = "/var/shared/telegraf/"

const PATH_GRAFANA string = "/var/shared/grafana/dashboards/"

func changeTelegrafDebug(instance string, debug int) error {
	// for enable debug we need to change telegraf.conf and set debug = false
	filePath := TELEGRAF_ROOT_PATH + instance + "/telegraf.conf"

	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		logger.Log.Errorf("Unable to open the file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	debugRegex := regexp.MustCompile(`^\s*debug\s*=\s*(true|false)\s*$`)
	var updatedLines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if debugRegex.MatchString(line) {
			if debug == 1 {
				line = "  debug = true"
			} else {
				line = "  debug = false"
			}
		}
		updatedLines = append(updatedLines, line)
	}

	if err := scanner.Err(); err != nil {
		logger.Log.Errorf("Error while reading the file %s: %v", filePath, err)
		return err
	}

	// Write the updated content back to the file
	err = os.WriteFile(filePath, []byte(strings.Join(updatedLines, "\n")), 0644)
	if err != nil {
		logger.Log.Errorf("Error while saving the file %s: %v", filePath, err)
		return err
	}

	return nil
}

func ManageDebug(instance string) error {

	// First retrieve the current debug state of the Instance
	var currentState int
	instance = strings.ToLower(instance)

	switch instance {
	case "MX":
		currentState = sqlite.ActiveAdmin.MXDebug
	case "PTX":
		currentState = sqlite.ActiveAdmin.PTXDebug
	case "ACX":
		currentState = sqlite.ActiveAdmin.ACXDebug
	case "EX":
		currentState = sqlite.ActiveAdmin.EXDebug
	case "QFX":
		currentState = sqlite.ActiveAdmin.QFXDebug
	case "SRX":
		currentState = sqlite.ActiveAdmin.SRXDebug
	case "CRPD":
		currentState = sqlite.ActiveAdmin.CRPDDebug
	case "CPTX":
		currentState = sqlite.ActiveAdmin.CPTXDebug
	case "VMX":
		currentState = sqlite.ActiveAdmin.VMXDebug
	case "VSRX":
		currentState = sqlite.ActiveAdmin.VSRXDebug
	case "VJUNOS":
		currentState = sqlite.ActiveAdmin.VJUNOSDebug
	case "VSWITCH":
		currentState = sqlite.ActiveAdmin.VSWITCHDebug
	case "VEVO":
		currentState = sqlite.ActiveAdmin.VEVODebug
	default:
		logger.Log.Errorf("Unsupported instance %s", instance)
		return errors.New("ManageDebug error: unsupported instance")
	}

	// Now modify the telegraf main config file of the instance
	if err := changeTelegrafDebug(instance, (currentState+1)%2); err != nil {
		logger.Log.Errorf("Error while changing debug mode in the intance %s", instance)
		return err
	}

	// Check if there is at least one router attached to the given instance
	atLeastOne := false
	// Retrive number of active routers per Telegraf instance
	for _, r := range sqlite.RtrList {
		if r.Family == instance {
			if r.Profile == 1 {
				atLeastOne = true
				break
			}
		}
	}

	if atLeastOne {
		// Now restart container only if there are active routers.
		if err := container.RestartContainer("telegraf_" + instance); err != nil {
			logger.Log.Errorf("Unable to restart containter telegraf_%s: %v", instance, err)
			// revert back to previous state
			changeTelegrafDebug(instance, currentState)
			return err
		}
	}

	// Save new State in DB
	if err := sqlite.UpdateDebugMode(instance, (currentState+1)%2); err != nil {
		logger.Log.Errorf("Unable to change debug state in DB for telegraf_%s: %v", instance, err)
		// revert back to previous state
		changeTelegrafDebug(instance, currentState)
		return err
	}

	if (currentState+1)%2 == 0 {
		logger.Log.Infof("Debug mode has been successfully disabled for the telegraf instance %s", instance)
	} else {
		logger.Log.Infof("Debug mode has been successfully enabled for the telegraf instance %s", instance)
	}

	return nil
}

func CheckVersion(searchVersion string, routerVersion string) bool {
	var operator string

	if len(searchVersion) > 2 {
		// Search operator
		if unicode.IsDigit(rune(searchVersion[0])) && unicode.IsDigit(rune(searchVersion[1])) {
			operator = "=="
		} else {
			operator = searchVersion[0:2]
			searchVersion = searchVersion[2:]
		}
		// Find out if routerVersion can be reduced
		r, _ := regexp.Compile(searchVersion + "*")
		result := r.FindString(routerVersion)
		if result != "" {
			routerVersion = result
		}
		switch operator {
		case "==":
			if strings.Compare(routerVersion, searchVersion) == 0 {
				return true
			}

		case ">>":
			if strings.Compare(routerVersion, searchVersion) > 0 {
				return true
			}
		case "<<":
			if strings.Compare(routerVersion, searchVersion) < 0 {
				return true

			}
		case ">=":
			if strings.Compare(routerVersion, searchVersion) >= 0 {
				return true
			}
		case "<=":
			if strings.Compare(routerVersion, searchVersion) <= 0 {
				return true
			}
		default:
			return false
		}
	}
	return false

}

func ConfigueStack(cfg *config.ConfigContainer, family string) error {

	logger.Log.Infof("Time to reconfigure JTS components for family %s", family)

	var temp *template.Template
	var families []string
	var readDirectory *os.File

	// create the slice for which families we have to reconfigure the stack
	if family == "all" {
		families = make([]string, 13)

		families[0] = "mx"
		families[1] = "ptx"
		families[2] = "acx"
		families[3] = "ex"
		families[4] = "qfx"
		families[5] = "srx"

		families[6] = "crpd"
		families[7] = "cptx"

		families[8] = "vmx"
		families[9] = "vsrx"
		families[10] = "vjunos"
		families[11] = "vswitch"
		families[12] = "vevo"

	} else {
		families = make([]string, 1)
		families[0] = family
	}

	// first create per type > per profile structure
	var cfgHierarchy map[string]map[string][]*sqlite.RtrEntry
	cfgHierarchy = make(map[string]map[string][]*sqlite.RtrEntry)
	for _, v := range sqlite.RtrList {
		if v.Profile == 1 {
			_, ok := cfgHierarchy[v.Family]
			if !ok {
				cfgHierarchy[v.Family] = make(map[string][]*sqlite.RtrEntry)
			}
			pfound := false
			var asso []string
			for _, v2 := range sqlite.AssoList {
				if v.Shortname == v2.Shortname {
					pfound = true
					asso = v2.Assos
				}
			}
			if pfound {
				for _, p := range asso {
					cfgHierarchy[v.Family][p] = append(cfgHierarchy[v.Family][p], v)
				}
			}
		}
	}

	// now recreate the telegraf config per family
	for _, f := range families {
		var directory string
		// remove all file of telegraf directory
		switch f {

		case "mx":
			readDirectory, _ = os.Open(PATH_MX)
			directory = PATH_MX
		case "ptx":
			readDirectory, _ = os.Open(PATH_PTX)
			directory = PATH_PTX
		case "acx":
			readDirectory, _ = os.Open(PATH_ACX)
			directory = PATH_ACX
		case "ex":
			readDirectory, _ = os.Open(PATH_EX)
			directory = PATH_EX
		case "qfx":
			readDirectory, _ = os.Open(PATH_QFX)
			directory = PATH_QFX
		case "srx":
			readDirectory, _ = os.Open(PATH_SRX)
			directory = PATH_SRX
		case "crpd":
			readDirectory, _ = os.Open(PATH_CRPD)
			directory = PATH_CRPD
		case "cptx":
			readDirectory, _ = os.Open(PATH_CPTX)
			directory = PATH_CPTX
		case "vmx":
			readDirectory, _ = os.Open(PATH_VMX)
			directory = PATH_VMX
		case "vsrx":
			readDirectory, _ = os.Open(PATH_VSRX)
			directory = PATH_VSRX
		case "vjunos":
			readDirectory, _ = os.Open(PATH_VJUNOS)
			directory = PATH_VJUNOS
		case "vswitch":
			readDirectory, _ = os.Open(PATH_VSWITCH)
			directory = PATH_VSWITCH
		case "vevo":
			readDirectory, _ = os.Open(PATH_VEVO)
			directory = PATH_VEVO
		}

		allFiles, _ := readDirectory.Readdir(0)

		for f := range allFiles {
			file := allFiles[f]

			fileName := file.Name()
			filePath := directory + fileName

			err := os.Remove(filePath)
			if err != nil {
				logger.Log.Errorf("Unable to clean the file %s: %v", filePath, err)
			}
		}

		// now parse all profiles of a given family
		for p, rtrs := range cfgHierarchy[f] {

			var filenames []Config
			perVersion := make(map[string][]*sqlite.RtrEntry)

			// extract definition of the profile
			switch f {

			case "mx":
				filenames = ActiveProfiles[p].Definition.TelCfg.MxCfg
			case "ptx":
				filenames = ActiveProfiles[p].Definition.TelCfg.PtxCfg
			case "acx":
				filenames = ActiveProfiles[p].Definition.TelCfg.AcxCfg
			case "ex":
				filenames = ActiveProfiles[p].Definition.TelCfg.ExCfg
			case "qfx":
				filenames = ActiveProfiles[p].Definition.TelCfg.QfxCfg
			case "srx":
				filenames = ActiveProfiles[p].Definition.TelCfg.SrxCfg
			case "crpd":
				filenames = ActiveProfiles[p].Definition.TelCfg.CrpdCfg
			case "cptx":
				filenames = ActiveProfiles[p].Definition.TelCfg.CptxCfg
			case "vmx":
				filenames = ActiveProfiles[p].Definition.TelCfg.VmxCfg
			case "vsrx":
				filenames = ActiveProfiles[p].Definition.TelCfg.VsrxCfg
			case "vjunos":
				filenames = ActiveProfiles[p].Definition.TelCfg.VjunosCfg
			case "vswitch":
				filenames = ActiveProfiles[p].Definition.TelCfg.VsrxCfg
			case "vevo":
				filenames = ActiveProfiles[p].Definition.TelCfg.VevoCfg
			}
			tls := false
			skip := false
			clienttls := false
			if sqlite.ActiveCred.UseTls == "yes" {
				tls = true
			}
			if sqlite.ActiveCred.SkipVerify == "yes" {
				skip = true
			}
			if sqlite.ActiveCred.ClientTls == "yes" {
				clienttls = true
			}

			// Create the map - per version > routers
			for _, r := range rtrs {
				confToApply := ""
				defaultConfig := ""

				for _, c := range filenames {
					// Save all config if present as a fallback solution if specific version not found
					if c.Version == "all" {
						defaultConfig = c.Config
					} else {
						result := CheckVersion(c.Version, r.Version)
						if result && (confToApply == "") {
							confToApply = c.Config
						}
					}
				}

				if confToApply != "" {
					perVersion[confToApply] = append(perVersion[confToApply], r)
					logger.Log.Infof("Router %s version %s will use configuration %s", r.Shortname, r.Version, confToApply)
				} else {
					if defaultConfig != "" {
						perVersion[defaultConfig] = append(perVersion[defaultConfig], r)
						logger.Log.Infof("Router %s  version %s will use configuration %s", r.Shortname, r.Version, defaultConfig)
					}
				}
			}

			for filename, v := range perVersion {

				rendRtrs := make([]string, 0)
				rendRtrsNet := make([]string, 0)
				for _, r := range v {
					rendRtrs = append(rendRtrs, r.Hostname+":"+strconv.Itoa(cfg.Gnmi.Port))
					rendRtrsNet = append(rendRtrsNet, r.Hostname)

				}
				// render profile
				t, err := template.ParseFiles("/var/active_profiles/" + p + "/" + filename)
				if err != nil {
					logger.Log.Errorf("Unable to open the telegraf file for rendering %s - err: %v", filename, err)
					continue
				}
				var mustErr error
				temp = template.Must(t, mustErr)
				if err != nil {
					logger.Log.Errorf("Unable to render file %s - err: %v", filename, err)
					continue
				}
				renderFile, err := os.Create(directory + filename)
				if err != nil {
					logger.Log.Errorf("Unable to open the target rendering file - err: %v", err)
					continue
				}
				defer renderFile.Close()
				err = temp.Execute(renderFile, map[string]interface{}{"rtrs": rendRtrs,
					"username":     sqlite.ActiveCred.GnmiUser,
					"password":     sqlite.ActiveCred.GnmiPwd,
					"tls":          tls,
					"skip":         skip,
					"tls_client":   clienttls,
					"usernetconf":  sqlite.ActiveCred.NetconfUser,
					"pwdnetconf":   sqlite.ActiveCred.NetconfPwd,
					"rtrs_netconf": rendRtrsNet})
				if err != nil {
					logger.Log.Errorf("Unable to write into render telegraf file - err: %v", err)
					continue
				}
			}
		}

	}

	// create the list of active profile dashboard name and copy the new version of each dashboard
	var excludeDash []string
	excludeDash = make([]string, 0)
	excludeDash = append(excludeDash, "home.json")
	for _, v := range cfgHierarchy {
		for p, _ := range v {
			for _, d := range ActiveProfiles[p].Definition.GrafaCfg {
				excludeDash = append(excludeDash, d)
				source, err := os.Open("/var/active_profiles/" + p + "/" + d) //open the source file
				if err != nil {
					logger.Log.Errorf("Unable to open the source dashboard %s - err: %v", d, err)
					continue
				}
				defer source.Close()
				destination, err := os.Create(PATH_GRAFANA + d) //create the destination file
				if err != nil {
					logger.Log.Errorf("Unable to open the destination dashboard %s - err: %v", d, err)
					continue
				}
				defer destination.Close()
				_, err = io.Copy(destination, source) //copy the contents of source to destination file
				if err != nil {
					logger.Log.Errorf("Unable to update the dashboard %s - err: %v", d, err)
					continue
				}
			}
		}
	}

	// Now clean grafana dashbord directory and keep only dashbords related to active profiles
	readDirectory, _ = os.Open(PATH_GRAFANA)
	allFiles, _ := readDirectory.Readdir(0)
	for f := range allFiles {
		file := allFiles[f]

		fileName := file.Name()
		filePath := PATH_GRAFANA + fileName

		//exclude home dashboard and active dashboards profile
		found := false
		for _, v := range excludeDash {
			if v == fileName {
				found = true
				break
			}
		}
		if !found {
			err := os.Remove(filePath)
			if err != nil {

				continue
			}
		}
	}

	// Create the list of Active Kapacitor script
	var kapaStart, kapaStop, kapaAll []string
	kapaStart = make([]string, 0)
	kapaStop = make([]string, 0)
	kapaAll = make([]string, 0)
	for _, v := range cfgHierarchy {
		for p, _ := range v {
			for _, d := range ActiveProfiles[p].Definition.KapaCfg {
				fileKapa := "/var/active_profiles/" + p + "/" + d
				to_add := true
				for _, a := range kapaAll {
					if a == fileKapa {
						to_add = false
						break
					}
				}
				if to_add {
					// kapaAll is to compare with ActiveTick later to delete unwanted tick scripts
					kapaAll = append(kapaAll, fileKapa)
				}
				found := false
				for i, _ := range kapacitor.ActiveTick {
					if i == fileKapa {
						found = true
						break
					}
				}
				// if kapa script not already active
				if !found {
					kapaStart = append(kapaStart, fileKapa)
				}
			}
		}
	}
	// check now those that need to be deleted
	for i, _ := range kapacitor.ActiveTick {
		found := false
		for _, v := range kapaAll {
			if i == v {
				found = true
				break
			}
		}
		if !found {
			kapaStop = append(kapaStop, i)
		}
	}
	// remove non active Kapascript
	kapacitor.DeleteTick(kapaStop)
	// Enable active scripts
	kapacitor.StartTick(kapaStart)

	// Restart grafana
	container.RestartContainer("grafana")

	// Restart telegraf instance(s)
	for _, f := range families {
		cntr := 0
		for _, rtrs := range cfgHierarchy[f] {
			cntr += len(rtrs)
		}

		// if cntr == 0 prefer shutdown the telegraf container
		if cntr == 0 {
			container.StopContainer("telegraf_" + f)
		} else {
			container.RestartContainer("telegraf_" + f)
		}
	}

	logger.Log.Infof("All JTS components reconfigured for family %s", family)
	return nil
}
