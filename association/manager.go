package association

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"jtso/logger"
	"os"
	"strings"
	"sync"

	"github.com/walle/targz"
)

type Config struct {
	Version string `json:"version"`
	Config  string `json:"conf"`
}

type Telegraf struct {
	VmxCfg []Config `json:"vmx"`
	MxCfg  []Config `json:"mx"`
	PtxCfg []Config `json:"ptx"`
	AcxCfg []Config `json:"acx"`
	QfxCfg []Config `json:"qfx"`
}

type DefProfile struct {
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

func PeriodicCheck() {
	logger.Log.Debug("Start periodic update of the profile db - scanning is starting")

	// reset current flag for Active profile
	ProfileLock.Lock()
	for k, v := range ActiveProfiles {
		v.Present = false
		ActiveProfiles[k] = v
	}

	// retrieve all tgz
	dir, err := os.Open("/var/profiles/")
	if err != nil {
		logger.Log.Errorf("Unable to open /var/active_profiles directory: %v", err)
		return
	}
	files, err := dir.ReadDir(0)
	if err != nil {
		logger.Log.Errorf("Unable to read /var/active_profiles directory: %v", err)
		return
	}

	for _, file := range files {
		if strings.Contains(file.Name(), "tgz") {
			filename := strings.Replace(file.Name(), ".tgz", "", -1)

			if _, ok := ActiveProfiles[filename]; ok {
				entry, _ := ActiveProfiles[filename]
				// existing profile - check if update
				// compute the hash of the file
				tmpFile, err := os.Open("/var/profiles/" + filename + ".tgz")
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
					err := os.RemoveAll("active_profile/" + filename + "/")
					if err != nil {
						logger.Log.Errorf("Unable to remove profile %s: %v", filename, err)
						continue
					}
					err = targz.Extract("/var/profiles/"+filename+".tgz", "/var/active_profiles/")
					if err != nil {
						logger.Log.Errorf("Unable to extract new profile %s: %v", filename, err)
						continue
					}

					// update definition JSON file
					jsonFile, err := os.Open("/var/active_profiles/" + filename + "/definition.json")
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

					// Copy cheatsheet image in the right assets directory
					source, err := os.Open("/var/active_profiles/" + filename + "/" + entry.Definition.Cheatsheet) //open the source file
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
				tmpFile, err := os.Open("/var/profiles/" + filename + ".tgz")
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

				err = targz.Extract("/var/profiles/"+filename+".tgz", "/var/active_profiles/")
				if err != nil {
					logger.Log.Errorf("Unable to extract new profile %s: %v", filename, err)
					continue
				}

				// open definition JSON file
				jsonFile, err := os.Open("/var/active_profiles/" + filename + "/definition.json")
				if err != nil {
					logger.Log.Errorf("Unable to open defintion.json for profile %s: %v", filename, err)
					continue
				}
				defer jsonFile.Close()

				byteValue, _ := io.ReadAll(jsonFile)

				// push json into definition structure
				json.Unmarshal(byteValue, entry.Definition)

				// Copy cheatsheet image in the right assets directory
				source, err := os.Open("/var/active_profiles/" + filename + "/" + entry.Definition.Cheatsheet) //open the source file
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
			err := os.RemoveAll("/var/active_profiles/" + v.Filename)
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
	logger.Log.Debug("End of the periodic update of the profiles db")
}
