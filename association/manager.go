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

type Telegraf struct {
	MxCfg  string `json:"mx"`
	PtxCfg string `json:"ptx"`
	AcxCfg string `json:"acx"`
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
	logger.Log.Info("Start periodic update of the profile - scanning is starting")

	// reset current flag for Active profile
	ProfileLock.Lock()
	for k, v := range ActiveProfiles {
		v.Present = false
		ActiveProfiles[k] = v
	}

	// retrieve all tgz
	dir, err := os.Open("profiles/")
	if err != nil {
		logger.Log.Errorf("Unable to open active_profiles directory: %v", err)
		return
	}
	files, err := dir.ReadDir(0)
	if err != nil {
		logger.Log.Errorf("Unable to read active_profiles directory: %v", err)
		return
	}

	for _, file := range files {
		filename := strings.Replace(file.Name(), ".tgz", "", -1)

		if _, ok := ActiveProfiles[filename]; ok {
			entry, _ := ActiveProfiles[filename]
			// existing profile - check if update
			// compute the hash of the file
			tmpFile, err := os.Open("profiles/" + filename + ".tgz")
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
				err = targz.Extract("profiles/"+filename+".tgz", "active_profiles/")
				if err != nil {
					logger.Log.Errorf("Unable to extract new profile %s: %v", filename, err)
					continue
				}

				// update definition JSON file
				jsonFile, err := os.Open("active_profiles/" + filename + "/definition.json")
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
			tmpFile, err := os.Open("profiles/" + filename + ".tgz")
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

			err = targz.Extract("profiles/"+filename+".tgz", "active_profiles/")
			if err != nil {
				logger.Log.Errorf("Unable to extract new profile %s: %v", filename, err)
				continue
			}

			// open definition JSON file
			jsonFile, err := os.Open("active_profiles/" + filename + "/definition.json")
			if err != nil {
				logger.Log.Errorf("Unable to open defintion.json for profile %s: %v", filename, err)
				continue
			}
			defer jsonFile.Close()

			byteValue, _ := io.ReadAll(jsonFile)

			// push json into definition structure
			json.Unmarshal(byteValue, entry.Definition)

			ActiveProfiles[filename] = entry
			logger.Log.Infof("New profile %s detected and added to active profiles", filename)
		}
	}

	// check now old profiles - where is no more tgz - clean them
	for k, v := range ActiveProfiles {
		if !v.Present {
			// Update profile
			err := os.RemoveAll("active_profiles/" + v.Filename)
			if err != nil {
				logger.Log.Errorf("Unable to remove profile %s: %v", v.Filename, err)
			}
			logger.Log.Infof("Legacy profile %s remove it", v.Filename)
			delete(ActiveProfiles, k)

		}
	}
	ProfileLock.Unlock()
	logger.Log.Info("End of the periodic update of the profile")
}
