package association

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"jtso/config"
	"jtso/logger"
	"jtso/sqlite"
	"jtso/worker"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/walle/targz"
)

type Config struct {
	Version string `json:"version"`
	Config  string `json:"conf"`
}

type Telegraf struct {
	MxCfg  []Config `json:"mx"`
	PtxCfg []Config `json:"ptx"`
	AcxCfg []Config `json:"acx"`
	ExCfg  []Config `json:"ex"`
	QfxCfg []Config `json:"qfx"`
	SrxCfg []Config `json:"srx"`

	CrpdCfg []Config `json:"crpd"`
	CptxCfg []Config `json:"cptx"`

	VmxCfg    []Config `json:"vmx"`
	VsrxCfg   []Config `json:"vsrx"`
	VjunosCfg []Config `json:"vjunos"`
	VevoCfg   []Config `json:"vevo"`
}

type DefProfile struct {
	Version     int      `json:"version"`
	Cheatsheet  string   `json:"cheatsheet"`
	Description string   `json:"description"`
	TelCfg      Telegraf `json:"telegraf"`
	KapaCfg     []string `json:"kapacitor"`
	GrafaCfg    []string `json:"grafana"`
}

type FileTgz struct {
	Filename   string
	Present    bool
	Hash       string
	Definition *DefProfile
}

var ActiveProfiles map[string]FileTgz
var ProfileLock *sync.Mutex

func init() {
	ActiveProfiles = make(map[string]FileTgz)
	ProfileLock = new(sync.Mutex)
}

func CleanActiveDirectory() error {
	entries, err := os.ReadDir(ACTIVE_PROFILES)
	if err != nil {
		logger.Log.Errorf("Unable to open %s directory: %v", ACTIVE_PROFILES, err)
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(ACTIVE_PROFILES, entry.Name())
		err := os.RemoveAll(entryPath)
		if err != nil {
			logger.Log.Errorf("Unable to remove %s: %v", entryPath, err)
			return err
		}
	}

	logger.Log.Infof("Directoy %s has been cleaned", ACTIVE_PROFILES)
	return nil

}
func PeriodicCheck(cfg *config.ConfigContainer) {
	logger.Log.Debug("Start periodic update of the profile db - scanning is starting")

	needRestart := make([]string, 0)

	// reset current flag for Active profile
	ProfileLock.Lock()
	for k, v := range ActiveProfiles {
		v.Present = false
		ActiveProfiles[k] = v
	}

	// retrieve all tgz
	dir, err := os.Open(PROFILES)
	if err != nil {
		logger.Log.Errorf("Unable to open %s directory: %v", PROFILES, err)
		return
	}
	files, err := dir.ReadDir(0)
	if err != nil {
		logger.Log.Errorf("Unable to read %s directory: %v", PROFILES, err)
		return
	}

	for _, file := range files {
		if strings.Contains(file.Name(), "tgz") {
			filename := strings.Replace(file.Name(), ".tgz", "", -1)

			entry, ok := ActiveProfiles[filename]

			if ok {
				// existing profile - check if update
				// compute the hash of the file
				tmpFile, err := os.Open(PROFILES + filename + ".tgz")
				if err != nil {
					logger.Log.Errorf("Unable to open file %s: %v", filename, err)
					continue
				}
				hash := md5.New()
				if _, err := io.Copy(hash, tmpFile); err != nil {
					logger.Log.Errorf("Unable to compute hash for file %s: %v", filename, err)
					continue
				}
				defer tmpFile.Close()
				hashInBytes := hash.Sum(nil)[:16]
				MD5String := hex.EncodeToString(hashInBytes)

				if ActiveProfiles[filename].Hash != MD5String {
					// Update profile
					err := os.RemoveAll(ACTIVE_PROFILES + filename + "/")
					if err != nil {
						logger.Log.Errorf("Unable to remove profile %s: %v", filename, err)
						continue
					}
					err = targz.Extract(PROFILES+filename+".tgz", ACTIVE_PROFILES)
					if err != nil {
						logger.Log.Errorf("Unable to extract new profile %s: %v", filename, err)
						continue
					}

					// update definition JSON file
					jsonFile, err := os.Open(ACTIVE_PROFILES + filename + "/definition.json")
					if err != nil {
						logger.Log.Errorf("Unable to open defintion.json for profile %s: %v", filename, err)
						continue
					}
					defer jsonFile.Close()

					byteValue, _ := io.ReadAll(jsonFile)
					entry.Definition = new(DefProfile)
					// push json into definition structure
					json.Unmarshal(byteValue, entry.Definition)
					entry.Hash = MD5String

					// Copy cheatsheet image in the right assets directories
					source, err := os.Open(ACTIVE_PROFILES + filename + "/" + entry.Definition.Cheatsheet) //open the source file
					if err != nil {
						logger.Log.Errorf("Unable to open the Cheatsheet file %s - err: %v", entry.Definition.Cheatsheet, err)
						continue
					}
					defer source.Close()
					destination, err := os.Create("html/assets/img/" + entry.Definition.Cheatsheet) //create the destination file
					if err != nil {
						logger.Log.Errorf("Unable to open the destination Cheatsheet %s - err: %v", entry.Definition.Cheatsheet, err)
						continue
					}
					defer destination.Close()
					_, err = io.Copy(destination, source) //copy the contents of source to destination file
					if err != nil {
						logger.Log.Errorf("Unable to update the Cheatsheet %s - err: %v", entry.Definition.Cheatsheet, err)
						continue
					}

					logger.Log.Infof("Profile %s has been updated", filename)
					for _, rtr := range sqlite.RtrList {
						if rtr.Profile == 0 {
							continue
						}
						for _, asso := range sqlite.AssoList {
							if asso.Shortname != rtr.Shortname {
								continue
							}
							for _, p := range asso.Assos {
								if p == filename {
									needRestart = append(needRestart, rtr.Family)
									goto NextRtr
								}
							}
						}
					NextRtr:
					}
				}

				entry.Present = true
				ActiveProfiles[filename] = entry

			} else {
				// new profile detected
				entry := FileTgz{}
				entry.Filename = filename
				entry.Present = true
				entry.Definition = new(DefProfile)

				// compute the hash of the file
				tmpFile, err := os.Open(PROFILES + filename + ".tgz")
				if err != nil {
					logger.Log.Errorf("Unable to open file %s: %v", filename, err)
					continue
				}
				hash := md5.New()
				if _, err := io.Copy(hash, tmpFile); err != nil {
					logger.Log.Errorf("Unable to compute hash for file %s: %v", filename, err)
					continue
				}
				defer tmpFile.Close()
				hashInBytes := hash.Sum(nil)[:16]
				MD5String := hex.EncodeToString(hashInBytes)
				entry.Hash = MD5String

				err = targz.Extract(PROFILES+filename+".tgz", ACTIVE_PROFILES)
				if err != nil {
					logger.Log.Errorf("Unable to extract new profile %s: %v", filename, err)
					continue
				}

				// open definition JSON file
				jsonFile, err := os.Open(ACTIVE_PROFILES + filename + "/definition.json")
				if err != nil {
					logger.Log.Errorf("Unable to open defintion.json for profile %s: %v", filename, err)
					continue
				}
				defer jsonFile.Close()

				byteValue, _ := io.ReadAll(jsonFile)

				// push json into definition structure
				json.Unmarshal(byteValue, entry.Definition)

				// Copy cheatsheet image in the right assets directories
				source, err := os.Open(ACTIVE_PROFILES + filename + "/" + entry.Definition.Cheatsheet) //open the source file
				if err != nil {
					logger.Log.Errorf("Unable to open the Cheatsheet file %s - err: %v", entry.Definition.Cheatsheet, err)
					continue
				}
				defer source.Close()
				destination, err := os.Create("html/assets/img/" + entry.Definition.Cheatsheet) //create the destination file
				if err != nil {
					logger.Log.Errorf("Unable to open the destination Cheatsheet %s - err: %v", entry.Definition.Cheatsheet, err)
					continue
				}
				defer destination.Close()
				_, err = io.Copy(destination, source) //copy the contents of source to destination file
				if err != nil {
					logger.Log.Errorf("Unable to update the Cheatsheet %s - err: %v", entry.Definition.Cheatsheet, err)
					continue
				}

				ActiveProfiles[filename] = entry
				logger.Log.Infof("New profile %s detected and added to active profiles", filename)
			}
		}
	}

	// check now old profiles - where is no more tgz - clean them
	for k, v := range ActiveProfiles {
		if !v.Present {
			// Update profile
			err := os.RemoveAll(ACTIVE_PROFILES + v.Filename)
			if err != nil {
				logger.Log.Errorf("Unable to remove profile %s: %v", v.Filename, err)
			}
			err = os.Remove("html/assets/img/" + v.Definition.Cheatsheet)
			if err != nil {
				logger.Log.Errorf("Unable to remove cheatsheet %s: %v", v.Definition.Cheatsheet, err)
			}
			logger.Log.Infof("Legacy profile %s remove it", v.Filename)
			delete(ActiveProfiles, k)

		}
	}
	ProfileLock.Unlock()
	if len(needRestart) > 0 {
		logger.Log.Info("Need to update the metadata...")
		go worker.Collect(cfg)

		for _, family := range needRestart {
			logger.Log.Infof("Need to restart the stack for %s family", family)
			go ConfigueStack(cfg, family)
		}
	}
	logger.Log.Debug("End of the periodic update of the profiles db")
}
