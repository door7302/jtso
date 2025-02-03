package association

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"jtso/config"
	"jtso/container"
	"jtso/kapacitor"
	"jtso/logger"
	"jtso/maker"
	"jtso/sqlite"
	"os"
	"regexp"
	"sort"
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
	case "mx":
		currentState = sqlite.ActiveAdmin.MXDebug
	case "ptx":
		currentState = sqlite.ActiveAdmin.PTXDebug
	case "acx":
		currentState = sqlite.ActiveAdmin.ACXDebug
	case "ex":
		currentState = sqlite.ActiveAdmin.EXDebug
	case "qfx":
		currentState = sqlite.ActiveAdmin.QFXDebug
	case "srx":
		currentState = sqlite.ActiveAdmin.SRXDebug
	case "crpd":
		currentState = sqlite.ActiveAdmin.CRPDDebug
	case "cptx":
		currentState = sqlite.ActiveAdmin.CPTXDebug
	case "vmx":
		currentState = sqlite.ActiveAdmin.VMXDebug
	case "vsrx":
		currentState = sqlite.ActiveAdmin.VSRXDebug
	case "vjunos":
		currentState = sqlite.ActiveAdmin.VJUNOSDebug
	case "vevo":
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
		families = make([]string, 12)

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
		families[11] = "vevo"

	} else {
		families = make([]string, 1)
		families[0] = family
	}
	// FIRST DEBUG ----------------------------------------------------------------------------

	// Map to store collections (family → collection → Collection struct)
	collections := make(map[string]map[string]sqlite.Collection)

	// Step 1: Build a lookup map for router profiles from AssoList
	routerProfiles := make(map[string][]string) // key: Shortname → value: Profile List
	for _, asso := range sqlite.AssoList {
		routerProfiles[asso.Shortname] = asso.Assos
	}

	// Step 2: Group routers by their profile sets
	profileSetToRouters := make(map[string][]*sqlite.RtrEntry)
	profileSetIndex := make(map[string]int) // Map unique profile sets to collection IDs
	collectionCounter := 1

	for _, rtr := range sqlite.RtrList {
		// Ignore routers with Profile = 0
		if rtr.Profile == 0 {
			continue
		}

		// Get the profiles from routerProfiles map
		profileKeys, exists := routerProfiles[rtr.Shortname]
		if !exists {
			continue // Skip if no profile association is found
		}

		// Sort profiles for uniqueness
		sort.Strings(profileKeys)

		// Create a unique profile key
		profileKey := fmt.Sprintf("%v", profileKeys)

		// Assign a collection ID if this profile set is new
		if _, exists := profileSetIndex[profileKey]; !exists {
			profileSetIndex[profileKey] = collectionCounter
			collectionCounter++
		}

		// Store the router in the corresponding profile set
		profileSetToRouters[profileKey] = append(profileSetToRouters[profileKey], rtr)
	}

	// Step 3: Construct the collections map
	for profileKey, routers := range profileSetToRouters {
		collectionID := "collect_" + strconv.Itoa(profileSetIndex[profileKey])

		// Extract actual profile slice
		profileSlice := []string{}
		fmt.Sscanf(profileKey, "%v", &profileSlice)

		// Get the family of the first router (all in the same family)
		family := routers[0].Family

		// Ensure family exists in the collections map
		if _, exists := collections[family]; !exists {
			collections[family] = make(map[string]sqlite.Collection)
		}

		// Assign to the collections map
		collections[family][collectionID] = sqlite.Collection{
			Profiles: profileSlice,
			Routers:  routers,
		}
	}

	for family, familyCollections := range collections {
		logger.Log.Info("Family:", family)
		for collectionID, collection := range familyCollections {
			logger.Log.Info("  ", collectionID, "=> Profiles:", collection.Profiles)
			logger.Log.Infof("     Routers:")
			for _, r := range collection.Routers {
				logger.Log.Info("       -", r.Hostname)
			}
		}
	}

	//// ----------------------------------------------------------------------------

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
				fullPath := "/var/active_profiles/" + p + "/" + filename

				rendRtrs := make([]string, 0)
				rendRtrsNet := make([]string, 0)
				for _, r := range v {
					rendRtrs = append(rendRtrs, r.Hostname+":"+strconv.Itoa(cfg.Gnmi.Port))
					rendRtrsNet = append(rendRtrsNet, r.Hostname)

				}

				// Check old model (raw file) vs new model (json)
				if strings.Contains(fullPath, ".json") {
					logger.Log.Info("New model detected")

					// new model
					newCfg, err := maker.LoadConfig(fullPath)
					if err != nil {
						continue
					}
					// Fill missing data
					if len(newCfg.GnmiList) > 0 {
						for i := range newCfg.GnmiList {
							newCfg.GnmiList[i].Rtrs = rendRtrs
							newCfg.GnmiList[i].Username = sqlite.ActiveCred.GnmiUser
							newCfg.GnmiList[i].Password = sqlite.ActiveCred.GnmiPwd
							newCfg.GnmiList[i].UseTls = tls
							newCfg.GnmiList[i].UseTlsClient = clienttls
							newCfg.GnmiList[i].SkipVerify = skip
						}
					}
					if len(newCfg.NetconfList) > 0 {
						for i := range newCfg.NetconfList {
							newCfg.NetconfList[i].Rtrs = rendRtrsNet
							newCfg.NetconfList[i].Username = sqlite.ActiveCred.NetconfUser
							newCfg.NetconfList[i].Password = sqlite.ActiveCred.NetconfPwd
						}
					}
					// render file
					payload, err := maker.RenderConf(newCfg)
					if err != nil {
						continue
					}
					newFileName := strings.TrimSuffix(filename, ".json") + ".conf"
					file, err := os.Create(directory + newFileName)
					if err != nil {
						logger.Log.Errorf("Unable to open the target rendering file %s - err: %v", newFileName, err)
						continue
					}
					defer file.Close()

					// Write text to the file
					_, err = file.WriteString(*payload)
					if err != nil {
						logger.Log.Errorf("Error writing to file %s: %v", newFileName, err)
						continue
					}

				} else {
					//old model
					// render profile
					t, err := template.ParseFiles(fullPath)
					if err != nil {
						logger.Log.Errorf("Unable to open the telegraf file for rendering %s - err: %v", filename, err)
						continue
					}
					var mustErr error
					temp = template.Must(t, mustErr)
					if mustErr != nil {
						logger.Log.Errorf("Unable to render file %s - err: %v", filename, mustErr)
						continue
					}
					renderFile, err := os.Create(directory + filename)
					if err != nil {
						logger.Log.Errorf("Unable to open the target rendering file %s - err: %v", filename, err)
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
