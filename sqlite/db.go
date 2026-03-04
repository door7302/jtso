package sqlite

import (
	"database/sql"
	"fmt"
	"jtso/influx"
	"jtso/logger"
	"jtso/security"
	"os"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type RtrEntry struct {
	Id        int
	Hostname  string
	Shortname string
	Family    string
	Model     string
	Version   string
	Profile   int
}

type Collection struct {
	ProfilesName []string
	ProfilesConf []string
	Routers      []*RtrEntry
}

type AssoEntry struct {
	Id        int
	Shortname string
	Assos     []string
}

type Cred struct {
	Id          int
	NetconfUser string
	NetconfPwd  string
	GnmiUser    string
	GnmiPwd     string
	UseTls      string
	SkipVerify  string
	ClientTls   string
	PasswordVer int // 0 = cleartext, 1 = encrypted
}

type Admin struct {
	Id int
	// HW devices
	MXDebug  int
	PTXDebug int
	ACXDebug int
	EXDebug  int
	QFXDebug int
	SRXDebug int
	// Native Container devices
	CRPDDebug int
	CPTXDebug int
	// VM devices
	VMXDebug    int
	VSRXDebug   int
	VJUNOSDebug int
	VEVODebug   int
	// On demand instance
	ONDEMANDDebug int
	// Influx retention policy (RP) duration
	RPDuration string
	//Ondemand config file name empty when stopped
	OndemandConfig string
}

type TelemetryInterval struct {
	Profile  string
	Path     string
	Mode     string
	Interval int
}

type CollectorParameters struct {
	Id                int
	MetricBatchSize   string
	MetricBufferLimit string
	FlushInterval     string
	FlushJitter       string
}

type KafkaConfig struct {
	Id          int
	Enabled     int
	Brokers     string
	Topic       string
	Format      string
	Version     string
	Compression int
	MessageSize int
}

var (
	db                        *sql.DB
	dbMu                      *sync.Mutex
	RtrList                   []*RtrEntry
	AssoList                  []*AssoEntry
	ActiveInterval            []*TelemetryInterval
	ActiveCred                Cred
	ActiveAdmin               Admin
	ActiveKafkaConfig         KafkaConfig
	ActiveCollectorParameters CollectorParameters
	SM                        *security.SecretManager
)

const SECRET_STORE string = "/data"

func Init(f string) error {
	var err error
	var secretChange bool
	err = nil
	dbMu = new(sync.Mutex)
	// check if db filename exist - if not create empty file
	if _, err := os.Stat(f); os.IsNotExist(err) {
		file, err := os.Create(f)
		if err != nil {
			logger.Log.Errorf("Error while creating DB file %s - err: %v", f, err)
			return err
		}
		defer file.Close()
		logger.Log.Infof("Initializing DB file %s - err: %v", f, err)
	}

	// open filename
	db, err = sql.Open("sqlite3", f)
	if err != nil {
		logger.Log.Infof("Error while opening DB %s - err: %v", f, err)
		return err
	}

	// Enable WAL
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		logger.Log.Infof("Error while enabling WAL for DB %s - err: %v", f, err)
	}

	// Initialize SecretManager
	SM, secretChange, err = security.NewSecretManager(SECRET_STORE)
	if err != nil {
		logger.Log.Errorf("Error initializing SecretManager: %v", err)
		return err
	}

	const createRtr string = `
		CREATE TABLE IF NOT EXISTS routers (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT,
		short TEXT,
		family TEXT,
		model TEXT,
		version TEXT,
		profile INTEGER
		);`

	const createAsso string = `
		CREATE TABLE IF NOT EXISTS associations (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT,
		listing TEXT
		);`

	const createCred string = `
		CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER NOT NULL PRIMARY KEY,
		netuser TEXT,
		netpwd TEXT,
		gnmiuser TEXT,
		gnmipwd TEXT,
		usetls TEXT,
		skipverify TEXT default "yes" NOT NULL,
		clienttls TEXT,
		passwordver INTEGER
		);`

	const createAdmin string = `
		CREATE TABLE IF NOT EXISTS administration (
		id INTEGER NOT NULL PRIMARY KEY,
		mxdebug INTEGER,
		ptxdebug INTEGER,
		acxdebug INTEGER,
		exdebug INTEGER,
		qfxdebug INTEGER,
		srxdebug INTEGER,
		crpddebug INTEGER,
		cptxdebug INTEGER,
		vmxdebug INTEGER,
		vsrxdebug INTEGER,
		vjunosdebug INTEGER,
		vevodebug INTEGER,
		ondemanddebug INTEGER,
		rpduration TEXT,
		ondemandconf TEXT
		);`

	const createTelegraf string = `
		CREATE TABLE IF NOT EXISTS telegraf (
		profile TEXT NOT NULL,
		path TEXT NOT NULL,
		mode TEXT,
		interval INTEGER,
		UNIQUE(profile, path)
		);`

	const createKafka string = `
		CREATE TABLE IF NOT EXISTS kafka_config (
		id INTEGER NOT NULL PRIMARY KEY,
		enabled INTEGER,
		brokers TEXT,
		topic TEXT,
		format TEXT,
		version TEXT,
		compression INTEGER,
		messagesize INTEGER
		);`

	const createCollector string = `
		CREATE TABLE IF NOT EXISTS collector_parameters (
		id INTEGER NOT NULL PRIMARY KEY,
		metric_batch_size TEXT,
		metric_buffer_limit TEXT,
		flush_interval TEXT,
		flush_jitter TEXT
		);`

	if _, err := db.Exec(createRtr); err != nil {
		logger.Log.Infof("Error while init DB %s Table routers - err: %v", f, err)
		return err
	}
	if _, err := db.Exec(createAsso); err != nil {
		logger.Log.Infof("Error while init DB %s Table associations - err: %v", f, err)
		return err
	}
	if _, err := db.Exec(createCred); err != nil {
		logger.Log.Infof("Error while init DB %s Table credentials - err: %v", f, err)
		return err
	}
	if _, err := db.Exec(createAdmin); err != nil {
		logger.Log.Infof("Error while init DB %s Table administration - err: %v", f, err)
		return err
	}
	if _, err := db.Exec(createTelegraf); err != nil {
		logger.Log.Infof("Error while init DB %s Table telegraf - err: %v", f, err)
		return err
	}
	if _, err := db.Exec(createKafka); err != nil {
		logger.Log.Infof("Error while init DB %s Table kafka_config - err: %v", f, err)
		return err
	}
	if _, err := db.Exec(createCollector); err != nil {
		logger.Log.Infof("Error while init DB %s Table collector_parameters - err: %v", f, err)
		return err
	}

	err = LoadAll(secretChange)
	return err
}

func GetRouterByShort(shortName string) (family string, name string, err error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	row := db.QueryRow("SELECT family, name FROM routers WHERE short=?", shortName)
	err = row.Scan(&family, &name)

	if err == sql.ErrNoRows {
		return "", "", err
	}
	if err != nil {
		logger.Log.Errorf("Error while querying router - err: %v", err)
		return "", "", err
	}

	return family, name, nil
}

func CheckAsso(n string) (bool, error) {
	dbMu.Lock()
	defer dbMu.Unlock()
	rows, err := db.Query("SELECT * FROM associations where name=?;", n)
	if err != nil {
		logger.Log.Errorf("Error while selecting associations - err: %v", err)
		return false, err
	}
	defer rows.Close()
	flag := rows.Next()
	return flag, nil
}

func GetTelegrafInterval(profile, path string) (interval int, found bool, err error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	row := db.QueryRow(`
		SELECT interval
		FROM telegraf
		WHERE profile = ? AND path = ?;
	`, profile, path)

	err = row.Scan(&interval)
	if err != nil {
		if err == sql.ErrNoRows {
			// Row does not exist
			return 0, false, nil
		}
		// Other error
		logger.Log.Errorf(
			"Error while querying telegraf interval (profile=%s, path=%s): %v",
			profile, path, err,
		)
		return 0, false, err
	}
	return interval, true, nil
}

func DeleteAllTelegrafByProfile(profile string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	res, err := db.Exec(`
		DELETE FROM telegraf
		WHERE profile = ?;
	`, profile)

	if err != nil {
		logger.Log.Errorf(
			"Error while deleting telegraf entries for profile '%s': %v",
			profile, err,
		)
		return err
	}

	rowsDeleted, _ := res.RowsAffected()
	logger.Log.Infof("Deleted %d telegraf entries for profile '%s'", rowsDeleted, profile)

	return loadAllInternal(false)
}

func UpdateKafkaConfig(enabled int, brokers, topic, format, version string, compression, messageSize int) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	_, err := db.Exec(`	
		INSERT INTO kafka_config (id, enabled, brokers, topic, format, version, compression, messagesize)
		VALUES (0, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			brokers = excluded.brokers,
			topic = excluded.topic,
			format = excluded.format,
			version = excluded.version,
			compression = excluded.compression,
			messagesize = excluded.messagesize;
	`, enabled, brokers, topic, format, version, compression, messageSize)

	if err != nil {
		logger.Log.Errorf("Error while upserting Kafka config: %v", err)
		return err
	}
	return loadAllInternal(false)
}

func UpdateCollectorParameters(metricBatchSize, metricBufferLimit, flushInterval, flushJitter string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err := db.Exec(`
		INSERT INTO collector_parameters (id, metric_batch_size, metric_buffer_limit, flush_interval, flush_jitter)
		VALUES (0, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			metric_batch_size = excluded.metric_batch_size,
			metric_buffer_limit = excluded.metric_buffer_limit,
			flush_interval = excluded.flush_interval,
			flush_jitter = excluded.flush_jitter;
	`, metricBatchSize, metricBufferLimit, flushInterval, flushJitter)

	if err != nil {
		logger.Log.Errorf("Error while upserting collector parameters: %v", err)
		return err
	}
	return loadAllInternal(false)
}

func UpdateInterval(profile, path, mode string, interval int) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	_, err := db.Exec(`
		INSERT INTO telegraf (profile, path, mode, interval)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(profile, path)
		DO UPDATE SET
			mode     = excluded.mode,
			interval = excluded.interval;
	`, profile, path, mode, interval)

	if err != nil {
		logger.Log.Errorf(
			"Error while upserting telegraf entry (profile=%s, path=%s): %v",
			profile, path, err,
		)
		return err
	}
	logger.Log.Infof("The interval for profile %s and path %s has been overridden with the value %d sec(s)", profile, path, interval)
	return loadAllInternal(false)
}

func DeleteInterval(profile, path string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	res, err := db.Exec(`
		DELETE FROM telegraf
		WHERE profile = ? AND path = ?;
	`, profile, path)

	if err != nil {
		logger.Log.Errorf(
			"Error while deleting telegraf entry (profile=%s, path=%s): %v",
			profile, path, err,
		)
		return err
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		logger.Log.Debugf(
			"No telegraf entry found to delete (profile=%s, path=%s)",
			profile, path,
		)
	}
	return loadAllInternal(false)
}

func AddRouter(n string, s string, f string, m string, v string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if _, err := db.Exec("INSERT INTO routers VALUES(NULL,?,?,?,?,?,?);", n, s, f, m, v, 0); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		return err
	}
	return loadAllInternal(false)
}

func DelAsso(n string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, err := db.Exec("DELETE FROM associations WHERE name=?;", n); err != nil {
		logger.Log.Errorf("Error while removing association for router %s - err: %v", n, err)
		return err
	}
	if _, err := db.Exec("UPDATE routers SET profile=? WHERE short=?;", 0, n); err != nil {
		logger.Log.Errorf("Error while updating router profile %s - err: %v", n, err)
		return err
	}
	return loadAllInternal(false)
}

func AddAsso(n string, a []string) error {

	dbMu.Lock()
	defer dbMu.Unlock()
	// convert list to string
	asso := strings.Join(a, "|")
	if _, err := db.Exec("INSERT INTO associations VALUES(NULL,?,?);", n, asso); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		return err
	}
	if _, err := db.Exec("UPDATE routers SET profile=? WHERE short=?;", 1, n); err != nil {
		logger.Log.Errorf("Error while updating router profile %s - err: %v", n, err)
		return err
	}
	return loadAllInternal(false)
}

func DelRouter(n string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, err := db.Exec("DELETE FROM routers WHERE short=?;", n); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		return err
	}
	return loadAllInternal(false)
}

func updateRouterProfile(n string, p int) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, err := db.Exec("UPDATE routers SET profile=? WHERE short=?;", p, n); err != nil {
		logger.Log.Errorf("Error while updating router profile %s - err: %v", n, err)
		return err
	}
	return loadAllInternal(false)
}

func UpdateRouter(s string, f string, m string, v string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, err := db.Exec("UPDATE routers SET family=?, model=?, version=? WHERE short=?", f, m, v, s); err != nil {
		logger.Log.Errorf("Error while updating router - err: %v", err)
		return err
	}
	return loadAllInternal(false)
}

func UpdateCredentials(nu string, np string, gu string, gp string, t string, s string, c string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	encNetPwd, err := security.Encrypt(SM.Current, np)
	if err != nil {
		logger.Log.Errorf("Error while encrypting netconf password - err: %v", err)
		return err
	}
	encGnmiPwd, err := security.Encrypt(SM.Current, gp)
	if err != nil {
		logger.Log.Errorf("Error while encrypting gnmi password - err: %v", err)
		return err
	}
	if _, err := db.Exec("UPDATE credentials SET netuser=?, netpwd=?, gnmiuser=?, gnmipwd=?, usetls=?, skipverify=?, clienttls=?  WHERE id=0;", nu, encNetPwd, gu, encGnmiPwd, t, s, c); err != nil {
		logger.Log.Errorf("Error while updating credential - err: %v", err)
		return err
	}
	return loadAllInternal(false)
}

func UpdateDebugMode(instance string, debug int) error {
	// Validate instance against allowlist to prevent SQL injection
	allowedInstances := map[string]bool{
		"mx": true, "ptx": true, "acx": true, "ex": true,
		"qfx": true, "srx": true, "crpd": true, "cptx": true,
		"vmx": true, "vsrx": true, "vjunos": true, "vevo": true,
		"ondemand": true,
	}
	if !allowedInstances[instance] {
		return fmt.Errorf("invalid instance name: %s", instance)
	}

	dbMu.Lock()
	defer dbMu.Unlock()
	// Save debug state
	debugInst := instance + "debug"

	// update the debug value for the instance
	if _, err := db.Exec("UPDATE administration SET "+debugInst+"=? WHERE id=0;", debug); err != nil {
		logger.Log.Errorf("Error while updating debug mode - err: %v", err)
		return err
	}
	return loadAllInternal(false)
}

func UpdateRpDuration(duration string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	// update the debug value for the instance
	if _, err := db.Exec("UPDATE administration SET rpduration=? WHERE id=0;", duration); err != nil {
		logger.Log.Errorf("Error while updating the RP duration - err: %v", err)
		return err
	}
	return loadAllInternal(false)
}

func LoadAll(secretRotation bool) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	return loadAllInternal(secretRotation)
}

// loadAllInternal performs the actual data reload without locking.
// Caller must hold dbMu.Lock().
func loadAllInternal(secretRotation bool) error {
	RtrList = make([]*RtrEntry, 0)
	rows, err := db.Query("SELECT * FROM routers;")
	if err != nil {
		logger.Log.Errorf("Error while selecting routers - err: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		i := RtrEntry{}
		err = rows.Scan(&i.Id, &i.Hostname, &i.Shortname, &i.Family, &i.Model, &i.Version, &i.Profile)
		if err != nil {
			logger.Log.Errorf("Error while parsing routers rows - err: %v", err)
			return err
		}
		RtrList = append(RtrList, &i)
	}

	AssoList = make([]*AssoEntry, 0)
	rows, err = db.Query("SELECT * FROM associations;")
	if err != nil {
		logger.Log.Errorf("Error while selecting associations - err: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		i := AssoEntry{}
		var tmpList string
		err = rows.Scan(&i.Id, &i.Shortname, &tmpList)
		if err != nil {
			logger.Log.Errorf("Error while parsing associations rows - err: %v", err)
			return err
		}
		// Fix legacy naming - be deprecated in future release
		tmpList = strings.ReplaceAll(tmpList, "power_extensive", "power")

		i.Assos = strings.Split(tmpList, "|")

		AssoList = append(AssoList, &i)
	}

	ActiveInterval = make([]*TelemetryInterval, 0)
	rows, err = db.Query("SELECT * FROM telegraf;")
	if err != nil {
		logger.Log.Errorf("Error while selecting telegraf - err: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		i := TelemetryInterval{}
		err = rows.Scan(&i.Profile, &i.Path, &i.Mode, &i.Interval)
		if err != nil {
			logger.Log.Errorf("Error while parsing telegraf interval rows - err: %v", err)
			return err
		}
		ActiveInterval = append(ActiveInterval, &i)
	}

	// Init with default credential in case of first launch and manage encryption if secret rotation or encryption just enabled
	encNetPwd, err := security.Encrypt(SM.Current, "lab123")
	if err != nil {
		logger.Log.Errorf("Error encrypting default netconf password - err: %v", err)
		return err
	}
	encGnmiPwd, err := security.Encrypt(SM.Current, "lab123")
	if err != nil {
		logger.Log.Errorf("Error encrypting default gnmi password - err: %v", err)
		return err
	}
	ActiveCred = Cred{Id: 0, NetconfUser: "lab", NetconfPwd: encNetPwd, GnmiUser: "lab", GnmiPwd: encGnmiPwd, UseTls: "no", SkipVerify: "yes", ClientTls: "no", PasswordVer: 1}

	rows, err = db.Query("SELECT * FROM credentials;")
	if err != nil {
		logger.Log.Errorf("Error while selecting credentials - err: %v", err)
		return err
	}
	defer rows.Close()
	i := rows.Next()
	if !i {
		// nothing in the DB regarding credential - add default one
		if _, err := db.Exec("INSERT INTO credentials VALUES(?,?,?,?,?,?,?,?,?);", 0, ActiveCred.NetconfUser, ActiveCred.NetconfPwd, ActiveCred.GnmiUser, ActiveCred.GnmiPwd, ActiveCred.UseTls, ActiveCred.SkipVerify, ActiveCred.ClientTls, ActiveCred.PasswordVer); err != nil {
			logger.Log.Errorf("Error while adding default credential - err: %v", err)
			return err
		}
	} else {
		colExists := false
		rows, err := db.Query("PRAGMA table_info(credentials);")
		if err != nil {
			logger.Log.Errorf("Error while checking table info - err: %v", err)
			return err
		}
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				logger.Log.Errorf("Error scanning table_info - err: %v", err)
				return err
			}
			if name == "passwordver" {
				colExists = true
				break
			}
		}
		rows.Close()
		if !colExists {
			_, err := db.Exec("ALTER TABLE credentials ADD COLUMN passwordver INTEGER DEFAULT 1;")
			if err != nil {
				logger.Log.Errorf("Error adding passwordver column - err: %v", err)
				return err
			}
			var netPwd, gnmiPwd string
			row := db.QueryRow("SELECT netpwd, gnmipwd FROM credentials WHERE id=0")
			err = row.Scan(&netPwd, &gnmiPwd)
			if err != nil {
				if err == sql.ErrNoRows {
					// No row exists in the table
					logger.Log.Errorf("No row exists in the table- err: %v", err)
					return err
				} else {
					// Some other DB error
					logger.Log.Errorf("Error scanning credentials - err: %v", err)
					return err
				}
			}
			encNetPwd, err := security.Encrypt(SM.Current, netPwd)
			if err != nil {
				logger.Log.Errorf("Error encrypting netconf password - err: %v", err)
				return err
			}
			encGnmiPwd, err := security.Encrypt(SM.Current, netPwd)
			if err != nil {
				logger.Log.Errorf("Error encrypting gnmi password - err: %v", err)
				return err
			}
			_, err = db.Exec("UPDATE credentials SET netpwd=?, gnmipwd=?, passwordver=1 WHERE id=0;", encNetPwd, encGnmiPwd)
			if err != nil {
				logger.Log.Errorf("Error encrypting existing passwords - err: %v", err)
				return err
			}
			logger.Log.Infof("Existing credentials have been encrypted and passwordver column has been added successfully")
		}
	}
	rows, err = db.Query("SELECT * FROM credentials;")
	if err != nil {
		logger.Log.Errorf("Error while selecting credentials - err: %v", err)
		return err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(
		&ActiveCred.Id,
		&ActiveCred.NetconfUser,
		&ActiveCred.NetconfPwd,
		&ActiveCred.GnmiUser,
		&ActiveCred.GnmiPwd,
		&ActiveCred.UseTls,
		&ActiveCred.SkipVerify,
		&ActiveCred.ClientTls,
		&ActiveCred.PasswordVer,
	)
	if err != nil {
		logger.Log.Errorf("Error while parsing credential rows - err: %v", err)
		return err
	}

	if secretRotation && ActiveCred.PasswordVer == 1 {
		// Try to decrypt with the new secret, if it fails try with the previous one
		netconfPwd, errNetconf := security.Decrypt(SM.Current, ActiveCred.NetconfPwd)
		if errNetconf != nil {
			netconfPwd, errNetconf = security.Decrypt(SM.Previous, ActiveCred.NetconfPwd)
			if errNetconf != nil {
				logger.Log.Errorf("Error decrypting netconf password with both current and previous secrets - err: %v", errNetconf)
				return errNetconf
			}
		}
		gnmiPwd, errGnmi := security.Decrypt(SM.Current, ActiveCred.GnmiPwd)
		if errGnmi != nil {
			gnmiPwd, errGnmi = security.Decrypt(SM.Previous, ActiveCred.GnmiPwd)
			if errGnmi != nil {
				logger.Log.Errorf("Error decrypting gnmi password with both current and previous secrets - err: %v", errGnmi)
				return errGnmi
			}
		}
		ActiveCred.NetconfPwd = netconfPwd
		ActiveCred.GnmiPwd = gnmiPwd

		// reencrypt with the new secret to update the DB and avoid keeping encrypted passwords with the previous secret
		encNetPwd, _ := security.Encrypt(SM.Current, netconfPwd)
		encGnmiPwd, _ := security.Encrypt(SM.Current, gnmiPwd)
		_, err = db.Exec("UPDATE credentials SET netpwd=?, gnmipwd=? WHERE id=0;", encNetPwd, encGnmiPwd)
		if err != nil {
			logger.Log.Errorf("Error re-encrypting passwords with the new secret - err: %v", err)
			return err
		}
		// Finalize the secret rotation by removing the previous secret from the SecretManager
		err = SM.Rotate()
		if err != nil {
			logger.Log.Errorf("Error finalizing secret rotation - err: %v", err)
			return err
		}
		logger.Log.Infof("Secret rotation has been finalized successfully, previous secret has been removed and credentials have been re-encrypted with the new secret")

	} else if ActiveCred.PasswordVer == 1 {
		netconfPwd, errNetconf := security.Decrypt(SM.Current, ActiveCred.NetconfPwd)
		if errNetconf != nil {
			logger.Log.Errorf("Error decrypting netconf password with current secret - err: %v", errNetconf)
			return errNetconf
		}
		gnmiPwd, errGnmi := security.Decrypt(SM.Current, ActiveCred.GnmiPwd)
		if errGnmi != nil {
			logger.Log.Errorf("Error decrypting gnmi password with current secret - err: %v", errGnmi)
			return errGnmi
		}
		ActiveCred.NetconfPwd = netconfPwd
		ActiveCred.GnmiPwd = gnmiPwd
	} else {
		logger.Log.Warnf("Credentials are stored in cleartext, consider rotating the secret to encrypt them")
	}

	ActiveAdmin = Admin{}
	rows, err = db.Query("SELECT * FROM administration;")
	if err != nil {
		logger.Log.Errorf("Error while selecting administration - err: %v", err)
		return err
	}
	defer rows.Close()
	i = rows.Next()
	if !i {
		// nothing in the DB regarding administration  - add default one
		if _, err := db.Exec("INSERT INTO administration VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);", 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, influx.DefaultRetention, ""); err != nil {
			logger.Log.Errorf("Error while adding default administration - err: %v", err)
			return err
		}
	} else {
		// Manage new fields: rpduration and ondemanddebug
		colExists, colExists2, colExists3 := false, false, false
		rows, err := db.Query("PRAGMA table_info(administration);")
		if err != nil {
			logger.Log.Errorf("Error while checking table info - err: %v", err)
			return err
		}
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				logger.Log.Errorf("Error scanning table_info - err: %v", err)
				return err
			}
			if name == "ondemanddebug" {
				colExists = true
			}
			if name == "rpduration" {
				colExists2 = true
			}
			if name == "ondemandconf" {
				colExists3 = true
			}

		}
		rows.Close()
		if !colExists {
			_, err := db.Exec("ALTER TABLE administration ADD COLUMN ondemanddebug INTEGER DEFAULT 0;")
			if err != nil {
				logger.Log.Errorf("Error adding ondemanddebug column - err: %v", err)
				return err
			}
		}
		if !colExists2 {
			_, err := db.Exec("ALTER TABLE administration ADD COLUMN rpduration TEXT DEFAULT '" + influx.DefaultRetention + "';")
			if err != nil {
				logger.Log.Errorf("Error adding rpduration column - err: %v", err)
				return err
			}
		}
		if !colExists3 {
			_, err := db.Exec("ALTER TABLE administration ADD COLUMN ondemandconf INTEGER DEFAULT 0;")
			if err != nil {
				logger.Log.Errorf("Error adding ondemandconf column - err: %v", err)
				return err
			}
		}
		// End of the specific piece of code managing new fields
	}
	rows, err = db.Query("SELECT * FROM administration;")
	if err != nil {
		logger.Log.Errorf("Error while selecting administration - err: %v", err)
		return err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(
		&ActiveAdmin.Id,
		&ActiveAdmin.MXDebug,
		&ActiveAdmin.PTXDebug,
		&ActiveAdmin.ACXDebug,
		&ActiveAdmin.EXDebug,
		&ActiveAdmin.QFXDebug,
		&ActiveAdmin.SRXDebug,
		&ActiveAdmin.CRPDDebug,
		&ActiveAdmin.CPTXDebug,
		&ActiveAdmin.VMXDebug,
		&ActiveAdmin.VSRXDebug,
		&ActiveAdmin.VJUNOSDebug,
		&ActiveAdmin.VEVODebug,
		&ActiveAdmin.ONDEMANDDebug,
		&ActiveAdmin.RPDuration,
		&ActiveAdmin.OndemandConfig,
	)
	if err != nil {
		logger.Log.Errorf("Error while parsing administration rows - err: %v", err)
		return err
	}

	ActiveKafkaConfig = KafkaConfig{}
	rows, err = db.Query("SELECT * FROM kafka_config;")
	if err != nil {
		logger.Log.Errorf("Error while selecting kafka_config - err: %v", err)
		return err
	}
	defer rows.Close()
	i = rows.Next()
	if i {
		err = rows.Scan(
			&ActiveKafkaConfig.Id,
			&ActiveKafkaConfig.Enabled,
			&ActiveKafkaConfig.Brokers,
			&ActiveKafkaConfig.Topic,
			&ActiveKafkaConfig.Format,
			&ActiveKafkaConfig.Version,
			&ActiveKafkaConfig.Compression,
			&ActiveKafkaConfig.MessageSize,
		)
		if err != nil {
			logger.Log.Errorf("Error while parsing kafka_config rows - err: %v", err)
			return err
		}
	} else {
		// nothing in the DB regarding kafka config - add default one
		if _, err := db.Exec("INSERT INTO kafka_config VALUES(?,?,?,?,?,?,?,?);", 0, 0, "localhost:9092", "jtso_topic", "json", "2.7.0", 0, 1000000); err != nil {
			logger.Log.Errorf("Error while adding default kafka config - err: %v", err)
			return err
		}
		ActiveKafkaConfig = KafkaConfig{Id: 0, Enabled: 0, Brokers: "localhost:9092", Topic: "jtso_topic", Format: "json", Version: "2.7.0", Compression: 0, MessageSize: 1000000}
	}

	ActiveCollectorParameters = CollectorParameters{}
	rows, err = db.Query("SELECT * FROM collector_parameters;")
	if err != nil {
		logger.Log.Errorf("Error while selecting collector_parameters - err: %v", err)
		return err
	}
	defer rows.Close()
	i = rows.Next()
	if i {
		err = rows.Scan(
			&ActiveCollectorParameters.Id,
			&ActiveCollectorParameters.MetricBatchSize,
			&ActiveCollectorParameters.MetricBufferLimit,
			&ActiveCollectorParameters.FlushInterval,
			&ActiveCollectorParameters.FlushJitter,
		)
		if err != nil {
			logger.Log.Errorf("Error while parsing collector_parameters rows - err: %v", err)
			return err
		}
	} else {
		// nothing in the DB regarding collector parameters - add default one
		if _, err := db.Exec("INSERT INTO collector_parameters VALUES(?,?,?,?,?);", 0, "5000", "100000", "5s", "0s"); err != nil {
			logger.Log.Errorf("Error while adding default collector parameters - err: %v", err)
			return err
		}
		ActiveCollectorParameters = CollectorParameters{Id: 0, MetricBatchSize: "5000", MetricBufferLimit: "100000", FlushInterval: "5s", FlushJitter: "0s"}
	}

	return nil
}

func CloseDb() error {
	logger.Log.Info("Closing database.")
	return db.Close()
}
