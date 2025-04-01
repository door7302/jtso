package association

import (
	"bufio"
	"errors"
	"fmt"
	"hash/fnv"
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
	"unicode"
)

const (
	TELEGRAF_ROOT_PATH string = "/var/shared/telegraf/"
	PATH_GRAFANA       string = "/var/shared/grafana/dashboards/"
	PROFILES           string = "/var/profiles/"
	ACTIVE_PROFILES    string = "/var/active_profiles/"
)

var PathMap = map[string]string{
	"mx":     "/var/shared/telegraf/mx/telegraf.d/",
	"ptx":    "/var/shared/telegraf/ptx/telegraf.d/",
	"acx":    "/var/shared/telegraf/acx/telegraf.d/",
	"ex":     "/var/shared/telegraf/ex/telegraf.d/",
	"qfx":    "/var/shared/telegraf/qfx/telegraf.d/",
	"srx":    "/var/shared/telegraf/srx/telegraf.d/",
	"crpd":   "/var/shared/telegraf/crpd/telegraf.d/",
	"cptx":   "/var/shared/telegraf/cptx/telegraf.d/",
	"vmx":    "/var/shared/telegraf/vmx/telegraf.d/",
	"vsrx":   "/var/shared/telegraf/vsrx/telegraf.d/",
	"vjunos": "/var/shared/telegraf/vjunos/telegraf.d/",
	"vevo":   "/var/shared/telegraf/vevo/telegraf.d/",
}

func hashStringFNV(input string) uint32 {
	hasher := fnv.New32a()
	hasher.Write([]byte(input))
	return hasher.Sum32()
}

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
		// PRefer Contains for ==
		// Find out if routerVersion can be reduced
		// r, _ := regexp.Compile(searchVersion + "*")
		// result := r.FindString(routerVersion)
		// if result != "" {
		//	routerVersion = result
		// }
		switch operator {
		case "==":
			if strings.Contains(routerVersion, searchVersion) {
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

	//var temp *template.Template
	var families []string

	profileSetToRouters := make(map[string][]*sqlite.RtrEntry)
	profileSetToProfilesFilename := make(map[string][]string)
	profileSetToProfilesName := make(map[string][]string)
	profileSetIndex := make(map[string]uint32)

	// Map to store collections (family → collection → Collection struct)
	collections := make(map[string]map[string]sqlite.Collection)

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

	// -----------------------------------------------------------------------------------------------------
	// Build a lookup map for router profiles from AssoList
	// -----------------------------------------------------------------------------------------------------
	routerProfiles := make(map[string][]string) // key: Shortname → value: Profile List
	for _, asso := range sqlite.AssoList {
		// Create a new slice with the same length as asso.Assos
		assosCopy := make([]string, len(asso.Assos))
		// Copy the contents of asso.Assos into the new slice
		copy(assosCopy, asso.Assos)
		routerProfiles[asso.Shortname] = assosCopy
	}

	// -----------------------------------------------------------------------------------------------------
	// Create the collection - based on Routers which are associated to profiles
	// -----------------------------------------------------------------------------------------------------
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

		// check if version is assigned to a profile and save file name
		profilesFilename := make([]string, len(profileKeys))
		profilesName := make([]string, len(profileKeys))
		for i, p := range profileKeys {

			// bypass unknown profile
			_, ok := ActiveProfiles[p]
			if !ok {
				logger.Log.Errorf("Collection issue - Unknown profile detected: %s - skip it", p)
				continue
			}
			profilesName[i] = p

			var filenameList []Config
			switch rtr.Family {
			case "mx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.MxCfg
			case "ptx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.PtxCfg
			case "acx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.AcxCfg
			case "ex":
				filenameList = ActiveProfiles[p].Definition.TelCfg.ExCfg
			case "qfx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.QfxCfg
			case "srx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.SrxCfg
			case "crpd":
				filenameList = ActiveProfiles[p].Definition.TelCfg.CrpdCfg
			case "cptx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.CptxCfg
			case "vmx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VmxCfg
			case "vsrx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VsrxCfg
			case "vjunos":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VjunosCfg
			case "vevo":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VevoCfg
			}

			// Check if a profile has as specific version for the given rtr
			savedVersion := ""
			profilesFilename[i] = ""

			for _, c := range filenameList {
				// Save all config if present as a fallback solution if specific version not found
				if c.Version == "all" && savedVersion == "" {
					profilesFilename[i] = c.Config
					savedVersion = "all"
				} else {
					result := CheckVersion(c.Version, rtr.Version)
					if result && (savedVersion == "" || savedVersion == "all") {
						profilesFilename[i] = c.Config
						savedVersion = c.Version
					}
				}
			}

			if savedVersion != "" {
				profileKeys[i] = p + "_" + savedVersion
			} else {
				// Reset entry if there is no filename found
				profileKeys[i] = ""
			}
		}

		// Sort profiles for uniqueness
		sort.Strings(profileKeys)

		// Create a unique profile key (string format for map indexing)
		profileKey := fmt.Sprintf("%s_%v", rtr.Family, profileKeys)

		// Store profile slice before hashing to prevent later issues
		if _, exists := profileSetIndex[profileKey]; !exists {
			profileSetIndex[profileKey] = hashStringFNV(profileKey)
			profileSetToProfilesFilename[profileKey] = profilesFilename
			profileSetToProfilesName[profileKey] = profilesName
		}

		// Store the router in the corresponding profile set
		profileSetToRouters[profileKey] = append(profileSetToRouters[profileKey], rtr)
	}

	// Finally the construction of  the collections map
	for profileKey, routers := range profileSetToRouters {
		// create the unit name
		collectionID := fmt.Sprintf("collection_%d", profileSetIndex[profileKey])

		// Retrieve original profile slice
		profilesFilename := profileSetToProfilesFilename[profileKey]
		profilesName := profileSetToProfilesName[profileKey]

		// Get the family of the first router (all in the same family)
		family := routers[0].Family

		// Ensure family exists in the collections map
		if _, exists := collections[family]; !exists {
			collections[family] = make(map[string]sqlite.Collection)
		}

		// Assign to the collections map
		collections[family][collectionID] = sqlite.Collection{
			ProfilesName: profilesName,
			ProfilesConf: profilesFilename,
			Routers:      routers,
		}

	}

	for family, familyCollections := range collections {
		logger.Log.Info("Update collections of Telegraf configs:")
		logger.Log.Infof(" Family: %s", family)
		for collectionID, collection := range familyCollections {
			logger.Log.Infof("  ID: %s:", collectionID)
			logger.Log.Info("     Profiles [files]:")
			for i, p := range collection.ProfilesConf {
				logger.Log.Infof("       -%s [%s]", collection.ProfilesName[i], p)
			}
			logger.Log.Info("     Routers part of the collection:")
			for _, r := range collection.Routers {
				logger.Log.Infof("       -%s", r.Hostname)
			}
		}
	}

	// -----------------------------------------------------------------------------------------------------
	// Now for each requested family - create the Telegraf optmized config
	// -----------------------------------------------------------------------------------------------------
	var telegrafCfgList []*maker.TelegrafConfig
	var readDirectory *os.File
	var err error
	for _, f := range families {
		path, exists := PathMap[f]
		if !exists {
			logger.Log.Errorf("Unknown router family: %s", f)
			continue
		}

		readDirectory, err = os.Open(path)
		if err != nil {
			logger.Log.Errorf("Unable to parse the folder %s: %v", path, err)
			continue
		}
		// clean the right directory only if there are files
		allFiles, _ := readDirectory.Readdir(0)

		for f := range allFiles {
			file := allFiles[f]

			fileName := file.Name()
			filePath := path + fileName

			err := os.Remove(filePath)
			if err != nil {
				logger.Log.Errorf("Unable to clean the file %s: %v", filePath, err)
			}
		}

		// For each collection
		for id, collection := range collections[f] {
			// create a new collection of config before optimisation
			telegrafCfgList = make([]*maker.TelegrafConfig, 0)
			for index, file := range collection.ProfilesConf {
				fullPath := ACTIVE_PROFILES + collection.ProfilesName[index] + "/" + file
				newCfg, err := maker.LoadConfig(fullPath)
				if err != nil {
					continue
				}
				telegrafCfgList = append(telegrafCfgList, newCfg)
			}

			// Create one unique config based on the list of configs
			mergedCfg := maker.OptimizeConf(telegrafCfgList)

			// Retrieve some common flags
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

			// create the list of rtr with port
			rendRtrs := make([]string, 0)
			rendRtrsNet := make([]string, 0)
			for _, r := range collection.Routers {
				rendRtrs = append(rendRtrs, r.Hostname+":"+strconv.Itoa(cfg.Gnmi.Port))
				rendRtrsNet = append(rendRtrsNet, r.Hostname)
			}

			// Fill missing data
			if len(mergedCfg.GnmiList) > 0 {
				for i := range mergedCfg.GnmiList {
					mergedCfg.GnmiList[i].Rtrs = rendRtrs
					mergedCfg.GnmiList[i].Username = sqlite.ActiveCred.GnmiUser
					mergedCfg.GnmiList[i].Password = sqlite.ActiveCred.GnmiPwd
					mergedCfg.GnmiList[i].UseTls = tls
					mergedCfg.GnmiList[i].UseTlsClient = clienttls
					mergedCfg.GnmiList[i].SkipVerify = skip
				}
			}
			if len(mergedCfg.NetconfList) > 0 {
				for i := range mergedCfg.NetconfList {
					mergedCfg.NetconfList[i].Rtrs = rendRtrsNet
					mergedCfg.NetconfList[i].Username = sqlite.ActiveCred.NetconfUser
					mergedCfg.NetconfList[i].Password = sqlite.ActiveCred.NetconfPwd
				}
			}
			// render file
			payload, err := maker.RenderConf(mergedCfg)
			if err != nil {
				continue
			}

			savedName := path + f + "_" + id + ".conf"
			file, err := os.Create(savedName)
			if err != nil {
				logger.Log.Errorf("Unable to open the target rendering file %s - err: %v", savedName, err)
				continue
			}
			defer file.Close()

			// Write text to the file
			_, err = file.WriteString(*payload)
			if err != nil {
				logger.Log.Errorf("Error writing to file %s: %v", savedName, err)
				continue
			}

		}

	}

	// -----------------------------------------------------------------------------------------------------
	// create the list of active profile dashboard name and copy the new version of each dashboard
	// -----------------------------------------------------------------------------------------------------
	var excludeDash []string
	excludeDash = make([]string, 0)
	excludeDash = append(excludeDash, "home.json")
	for _, v := range collections {
		for _, c := range v {
			for _, p := range c.ProfilesName {
				// bypass unknown profile
				_, ok := ActiveProfiles[p]
				if !ok {
					logger.Log.Errorf("Grafana update - Unknown profile detected: %s - skip it", p)
					continue
				}
				for _, d := range ActiveProfiles[p].Definition.GrafaCfg {
					excludeDash = append(excludeDash, d)
					source, err := os.Open(ACTIVE_PROFILES + p + "/" + d) //open the source file
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
					logger.Log.Infof("Active dashboard %s for profile %s", d, p)
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

		// exclude home dashboard and active dashboards profile
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

	// -----------------------------------------------------------------------------------------------------
	// Create the list of Active Kapacitor script
	// -----------------------------------------------------------------------------------------------------
	var kapaStart, kapaStop, kapaAll []string
	kapaStart = make([]string, 0)
	kapaStop = make([]string, 0)
	kapaAll = make([]string, 0)
	for _, v := range collections {
		for _, c := range v {
			for _, p := range c.ProfilesName {
				// bypass unknown profile
				_, ok := ActiveProfiles[p]
				if !ok {
					logger.Log.Errorf("Kapacitor update - Unknown profile detected: %s - skip it", p)
					continue
				}
				for _, d := range ActiveProfiles[p].Definition.KapaCfg {
					fileKapa := ACTIVE_PROFILES + p + "/" + d
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
	if len(kapaStop) > 0 {
		kapacitor.DeleteTick(kapaStop)
	}

	// Enable active scripts
	if len(kapaStart) > 0 {
		kapacitor.StartTick(kapaStart)
	}

	// Restart grafana
	container.RestartContainer("grafana")

	// -----------------------------------------------------------------------------------------------------
	// Restart telegraf instance(s) : only for the affected families
	// -----------------------------------------------------------------------------------------------------
	for _, f := range families {
		cntr := 0
		for _, c := range collections[f] {
			cntr += len(c.Routers)
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
