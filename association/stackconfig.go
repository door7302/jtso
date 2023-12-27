package association

import (
	"context"
	"io"
	"jtso/config"
	"jtso/kapacitor"
	"jtso/logger"
	"jtso/sqlite"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const PATH_VMX string = "/var/shared/telegraf/vmx/telegraf.d/"
const PATH_MX string = "/var/shared/telegraf/mx/telegraf.d/"
const PATH_PTX string = "/var/shared//telegraf/ptx/telegraf.d/"
const PATH_ACX string = "/var/shared//telegraf/acx/telegraf.d/"
const PATH_GRAFANA string = "/var/shared/grafana/dashboards/"

func ConfigueStack(cfg *config.ConfigContainer, family string) error {

	logger.Log.Infof("Time to reconfigure JTS components for family %s", family)

	var temp *template.Template
	var familes []string
	var readDirectory *os.File

	// create the slice for which families we have to reconfigure the stack
	if family == "all" {
		familes = make([]string, 4)
		familes[0] = "vmx"
		familes[1] = "mx"
		familes[2] = "ptx"
		familes[3] = "acx"
	} else {
		familes = make([]string, 1)
		familes[0] = family
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
	for _, f := range familes {
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
			var perVersion map[string][]*sqlite.RtrEntry
			perVersion = make(map[string][]*sqlite.RtrEntry)

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
				keep_version := "all"
				save_conf := ""
				for _, c := range filenames {
					if c.Version != "all" {
						comp := strings.Compare(r.Version, c.Version)
						if comp == -1 {
							if keep_version != "all" {
								comp := strings.Compare(c.Version, keep_version)
								if comp == -1 {
									keep_version = c.Version
									save_conf = c.Config
								}
							} else {
								keep_version = c.Version
								save_conf = c.Config
							}
						}
					} else {
						save_conf = c.Config
					}
				}
				perVersion[save_conf] = append(perVersion[save_conf], r)
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
	var kapaStart, kapaStop []string
	kapaStart = make([]string, 0)
	kapaStop = make([]string, 0)
	logger.Log.Errorf("Len de Active tick: %d", len(kapacitor.ActiveTick))
	for _, v := range cfgHierarchy {
		for p, _ := range v {
			for _, d := range ActiveProfiles[p].Definition.KapaCfg {
				fileKapa := "/var/active_profiles/" + p + "/" + d
				found := false
				for i, _ := range kapacitor.ActiveTick {
					logger.Log.Errorf("TOTO: %s - %s", i, fileKapa)
					if i == fileKapa {
						found = true
						break
					}
				}
				// if kapa script not already active
				if !found {
					logger.Log.Errorf("ADD to start: %s", fileKapa)
					kapaStart = append(kapaStart, fileKapa)
				}
			}
		}
	}
	// check now those that need to be deleted
	for i, _ := range kapacitor.ActiveTick {
		found := false
		for _, v := range kapaStart {
			logger.Log.Errorf("DEBUG: %s - %s", i, v)
			if i == v {
				found = true
				break
			}
		}
		if !found {
			logger.Log.Errorf("Add to stop: %s", i)
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
	for _, f := range familes {
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
