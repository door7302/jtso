package association

import (
	"context"
	"io"
	"jtso/config"
	"jtso/kapacitor"
	"jtso/logger"
	"jtso/sqlite"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const PATH_VMX string = "/var/shared/telegraf/vmx/telegraf.d/"
const PATH_MX string = "/var/shared/telegraf/mx/telegraf.d/"
const PATH_PTX string = "/var/shared//telegraf/ptx/telegraf.d/"
const PATH_ACX string = "/var/shared//telegraf/acx/telegraf.d/"
const PATH_GRAFANA string = "/var/shared/grafana/dashboards/"

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
		families = make([]string, 4)
		families[0] = "vmx"
		families[1] = "mx"
		families[2] = "ptx"
		families[3] = "acx"
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
		case "vmx":
			readDirectory, _ = os.Open(PATH_VMX)
			directory = PATH_VMX
		case "mx":
			readDirectory, _ = os.Open(PATH_MX)
			directory = PATH_MX
		case "ptx":
			readDirectory, _ = os.Open(PATH_PTX)
			directory = PATH_PTX
		case "acx":
			readDirectory, _ = os.Open(PATH_ACX)
			directory = PATH_ACX
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
			case "vmx":
				filenames = ActiveProfiles[p].Definition.TelCfg.VmxCfg
			case "mx":
				filenames = ActiveProfiles[p].Definition.TelCfg.MxCfg
			case "ptx":
				filenames = ActiveProfiles[p].Definition.TelCfg.PtxCfg
			case "acx":
				filenames = ActiveProfiles[p].Definition.TelCfg.AcxCfg
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
				for _, r := range v {
					rendRtrs = append(rendRtrs, r.Hostname+":"+strconv.Itoa(cfg.Gnmi.Port))

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
				err = temp.Execute(renderFile, map[string]interface{}{"rtrs": rendRtrs, "username": sqlite.ActiveCred.GnmiUser, "password": sqlite.ActiveCred.GnmiPwd, "tls": tls, "skip": skip, "tls_client": clienttls})
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

	// restart Containers :
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
		return err
	}
	defer cli.Close()

	timeout := 10

	// Restart grafana
	err = cli.ContainerRestart(context.Background(), "grafana", container.StopOptions{Signal: "SIGTERM", Timeout: &timeout})
	if err != nil {
		logger.Log.Errorf("Unable to restart Grafana container: %v", err)
		return err
	}
	logger.Log.Info("Grafana container has been restarted")

	// Restart telegraf instance(s)
	for _, f := range families {
		cntr := 0
		for _, rtrs := range cfgHierarchy[f] {
			cntr += len(rtrs)
		}

		// if cntr == 0 prefer shutdown the telegraf container
		if cntr == 0 {
			err = cli.ContainerStop(context.Background(), "telegraf_"+f, container.StopOptions{Signal: "SIGTERM", Timeout: &timeout})
			if err != nil {
				logger.Log.Errorf("Unable to stop telegraf_"+f+" container: %v", err)
				continue
			}
			logger.Log.Info("telegraf_" + f + " container has been stopped - no more router attached")
		} else {
			err = cli.ContainerRestart(context.Background(), "telegraf_"+f, container.StopOptions{Signal: "SIGTERM", Timeout: &timeout})
			if err != nil {
				logger.Log.Errorf("Unable to restart telegraf_"+f+" container: %v", err)
				continue
			}
			logger.Log.Info("telegraf_" + f + " container has been restarted")
		}
	}

	logger.Log.Infof("All JTS components reconfigured for family %s", family)
	return nil
}
