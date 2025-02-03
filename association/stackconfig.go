package association

import (
	"bufio"
	"errors"
	"fmt"
	"hash/fnv"
	"jtso/config"
	"jtso/container"
	"jtso/logger"
	"jtso/sqlite"
	"os"
	"regexp"
	"sort"
	"strings"
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

	// Build a lookup map for router profiles from AssoList
	routerProfiles := make(map[string][]string) // key: Shortname → value: Profile List
	for _, asso := range sqlite.AssoList {
		// Create a new slice with the same length as asso.Assos
		assosCopy := make([]string, len(asso.Assos))
		// Copy the contents of asso.Assos into the new slice
		copy(assosCopy, asso.Assos)
		routerProfiles[asso.Shortname] = assosCopy
	}

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

			var filenameList []Config
			var directory string
			var err error
			var readDirectory *os.File

			profilesName[i] = p

			switch rtr.Family {
			case "mx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.MxCfg
				readDirectory, err = os.Open(PATH_MX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_MX, err)
					continue
				}
				directory = PATH_MX
			case "ptx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.PtxCfg
				readDirectory, err = os.Open(PATH_PTX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_PTX, err)
					continue
				}
				directory = PATH_PTX
			case "acx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.AcxCfg
				readDirectory, err = os.Open(PATH_ACX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_ACX, err)
					continue
				}
				directory = PATH_ACX
			case "ex":
				filenameList = ActiveProfiles[p].Definition.TelCfg.ExCfg
				readDirectory, err = os.Open(PATH_EX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_EX, err)
					continue
				}
				directory = PATH_EX

			case "qfx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.QfxCfg
				readDirectory, err = os.Open(PATH_QFX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_QFX, err)
					continue
				}
				directory = PATH_QFX
			case "srx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.SrxCfg
				readDirectory, err = os.Open(PATH_SRX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_SRX, err)
					continue
				}
				directory = PATH_SRX
			case "crpd":
				filenameList = ActiveProfiles[p].Definition.TelCfg.CrpdCfg
				readDirectory, err = os.Open(PATH_CRPD)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_CRPD, err)
					continue
				}
				directory = PATH_CRPD
			case "cptx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.CptxCfg
				readDirectory, err = os.Open(PATH_CPTX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_CPTX, err)
					continue
				}
				directory = PATH_CPTX
			case "vmx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VmxCfg
				readDirectory, err = os.Open(PATH_VMX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_VMX, err)
					continue
				}
				directory = PATH_VMX
			case "vsrx":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VsrxCfg
				readDirectory, err = os.Open(PATH_VSRX)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_VSRX, err)
					continue
				}
				directory = PATH_VSRX
			case "vjunos":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VjunosCfg
				readDirectory, err = os.Open(PATH_VJUNOS)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_VJUNOS, err)
					continue
				}
				directory = PATH_VJUNOS
			case "vevo":
				filenameList = ActiveProfiles[p].Definition.TelCfg.VevoCfg
				readDirectory, err = os.Open(PATH_VEVO)
				if err != nil {
					logger.Log.Errorf("Unable to parse the folder %s: %v", PATH_VEVO, err)
					continue
				}
				directory = PATH_VEVO
			}

			// clean the right directory only if there are files
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

			// Check if a profile has as specific version for the given rtr
			savedVersion := ""
			profilesFilename[i] = ""

			for _, c := range filenameList {
				// Save all config if present as a fallback solution if specific version not found
				logger.Log.Infof("Config: %v  -   C.Ver %v; R.Ver %v", c.Config, c.Version, rtr.Version)
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
		profileKey := fmt.Sprintf("%v", profileKeys)

		// Store profile slice before hashing to prevent later issues
		if _, exists := profileSetIndex[profileKey]; !exists {
			profileSetIndex[profileKey] = hashStringFNV(rtr.Family + profileKey)
			profileSetToProfilesFilename[profileKey] = profilesFilename
			profileSetToProfilesName[profileKey] = profilesName
		}

		// Store the router in the corresponding profile set
		profileSetToRouters[profileKey] = append(profileSetToRouters[profileKey], rtr)
	}

	// Step 3: Construct the collections map
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

	// DEBUG ----------------------------------------------------------------------------
	for family, familyCollections := range collections {
		logger.Log.Info("Family:", family)
		for collectionID, collection := range familyCollections {
			logger.Log.Info("  ", collectionID, "=> Profiles:")
			for i, p := range collection.ProfilesConf {
				logger.Log.Info("     >", collection.ProfilesName[i], " : ", p)
			}
			logger.Log.Infof("     Routers:")
			for _, r := range collection.Routers {
				logger.Log.Info("       -", r.Hostname)
			}
		}
	}
	//// ----------------------------------------------------------------------------

	logger.Log.Infof("All JTS components reconfigured for family %s", family)
	return nil
}
