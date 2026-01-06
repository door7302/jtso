package sqlite

import (
	"database/sql"
	"jtso/influx"
	"jtso/logger"
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
	// Influx retention policy (RP) duration
	RPDuration string
}

type TelemetryInterval struct {
	Profile  string
	Path     string
	Mode     string
	Interval int
}

var db *sql.DB
var dbMu *sync.Mutex
var RtrList []*RtrEntry
var AssoList []*AssoEntry
var ActiveInterval []*TelemetryInterval
var ActiveCred Cred
var ActiveAdmin Admin

func Init(f string) error {
	var err error
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
		clienttls TEXT
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
		rpduration TEXT
		);`

	const createTelegraf string = `
		CREATE TABLE IF NOT EXISTS telegraf (
		profile TEXT NOT NULL,
		path TEXT NOT NULL,
		mode TEXT,
		interval INTEGER,
		UNIQUE(profile, path)
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
	err = LoadAll()
	return err
}

func CheckAsso(n string) (bool, error) {
	dbMu.Lock()
	rows, err := db.Query("SELECT * FROM associations where name=?;", n)
	if err != nil {
		logger.Log.Errorf("Error while selecting associations - err: %v", err)
		dbMu.Unlock()
		return false, err
	}
	defer rows.Close()
	flag := rows.Next()
	dbMu.Unlock()
	return flag, nil
}

func GetTelegrafInterval(profile, path string) (interval int, found bool, err error) {
	dbMu.Lock()

	row := db.QueryRow(`
		SELECT interval
		FROM telegraf
		WHERE profile = ? AND path = ?;
	`, profile, path)

	err = row.Scan(&interval)
	if err != nil {
		if err == sql.ErrNoRows {
			// Row does not exist
			dbMu.Unlock()
			return 0, false, nil
		}
		// Other error
		logger.Log.Errorf(
			"Error while querying telegraf interval (profile=%s, path=%s): %v",
			profile, path, err,
		)
		dbMu.Unlock()
		return 0, false, err
	}
	dbMu.Unlock()
	return interval, true, nil
}

func DeleteAllTelegrafByProfile(profile string) error {
	dbMu.Lock()

	res, err := db.Exec(`
		DELETE FROM telegraf
		WHERE profile = ?;
	`, profile)

	if err != nil {
		logger.Log.Errorf(
			"Error while deleting telegraf entries for profile '%s': %v",
			profile, err,
		)
		dbMu.Unlock()
		return err
	}

	rowsDeleted, _ := res.RowsAffected()
	logger.Log.Infof("Deleted %d telegraf entries for profile '%s'", rowsDeleted, profile)

	dbMu.Unlock()
	return LoadAll()
}

func UpdateInterval(profile, path, mode string, interval int) error {
	dbMu.Lock()

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
		dbMu.Unlock()
		return err
	}
	logger.Log.Infof("The interval for profile %s and path %s has been overridden with the value %d sec(s)", profile, path, interval)
	dbMu.Unlock()
	return LoadAll()
}

func DeleteInterval(profile, path string) error {
	dbMu.Lock()

	res, err := db.Exec(`
		DELETE FROM telegraf
		WHERE profile = ? AND path = ?;
	`, profile, path)

	if err != nil {
		logger.Log.Errorf(
			"Error while deleting telegraf entry (profile=%s, path=%s): %v",
			profile, path, err,
		)
		dbMu.Unlock()
		return err
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		logger.Log.Debugf(
			"No telegraf entry found to delete (profile=%s, path=%s)",
			profile, path,
		)
	}
	dbMu.Unlock()
	return LoadAll()
}

func AddRouter(n string, s string, f string, m string, v string) error {
	dbMu.Lock()

	if _, err := db.Exec("INSERT INTO routers VALUES(NULL,?,?,?,?,?,?);", n, s, f, m, v, 0); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func DelAsso(n string) error {
	dbMu.Lock()
	if _, err := db.Exec("DELETE FROM associations WHERE name=?;", n); err != nil {
		logger.Log.Errorf("Error while removing association for router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := updateRouterProfile(n, 0)
	return err
}

func AddAsso(n string, a []string) error {

	dbMu.Lock()
	// convert list to string
	var asso string
	for i, v := range a {
		if i != len(a)-1 {
			asso += v + "|"
		} else {
			asso += v
		}
	}
	if _, err := db.Exec("INSERT INTO associations VALUES(NULL,?,?);", n, asso); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := updateRouterProfile(n, 1)
	return err
}

func DelRouter(n string) error {
	dbMu.Lock()
	if _, err := db.Exec("DELETE FROM routers WHERE short=?;", n); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func updateRouterProfile(n string, p int) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE routers SET profile=? WHERE short=?;", p, n); err != nil {
		logger.Log.Errorf("Error while updating router profile %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func UpdateRouter(s string, f string, m string, v string) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE routers SET family=?, model=?, version=? WHERE short=?", f, m, v, s); err != nil {
		logger.Log.Errorf("Error while updating router - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func UpdateCredentials(nu string, np string, gu string, gp string, t string, s string, c string) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE credentials SET netuser=?, netpwd=?, gnmiuser=?, gnmipwd=?, usetls=?, skipverify=?, clienttls=?  WHERE id=0;", nu, np, gu, gp, t, s, c); err != nil {
		logger.Log.Errorf("Error while updating credential - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func UpdateDebugMode(instance string, debug int) error {
	dbMu.Lock()
	// Save debug state
	debugInst := instance + "debug"

	// update the debug value for the instance
	if _, err := db.Exec("UPDATE administration SET "+debugInst+"=? WHERE id=0;", debug); err != nil {
		logger.Log.Errorf("Error while updating debug mode - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func UpdateRpDuration(duration string) error {
	dbMu.Lock()
	// update the debug value for the instance
	if _, err := db.Exec("UPDATE administration SET rpduration=? WHERE id=0;", duration); err != nil {
		logger.Log.Errorf("Error while updating the RP duration - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	return LoadAll()
}

func LoadAll() error {
	dbMu.Lock()
	RtrList = make([]*RtrEntry, 0)
	rows, err := db.Query("SELECT * FROM routers;")
	if err != nil {
		logger.Log.Errorf("Error while selecting routers - err: %v", err)
		dbMu.Unlock()
		return err
	}
	defer rows.Close()
	for rows.Next() {
		i := RtrEntry{}
		err = rows.Scan(&i.Id, &i.Hostname, &i.Shortname, &i.Family, &i.Model, &i.Version, &i.Profile)
		if err != nil {
			logger.Log.Errorf("Error while parsing routers rows - err: %v", err)
			dbMu.Unlock()
			return err
		}
		RtrList = append(RtrList, &i)
	}

	AssoList = make([]*AssoEntry, 0)
	rows, err = db.Query("SELECT * FROM associations;")
	if err != nil {
		logger.Log.Errorf("Error while selecting associations - err: %v", err)
		dbMu.Unlock()
		return err
	}
	defer rows.Close()
	for rows.Next() {
		i := AssoEntry{}
		var tmpList string
		err = rows.Scan(&i.Id, &i.Shortname, &tmpList)
		if err != nil {
			logger.Log.Errorf("Error while parsing associations rows - err: %v", err)
			dbMu.Unlock()
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
		dbMu.Unlock()
		return err
	}
	defer rows.Close()
	for rows.Next() {
		i := TelemetryInterval{}
		err = rows.Scan(&i.Profile, &i.Path, &i.Mode, &i.Interval)
		if err != nil {
			logger.Log.Errorf("Error while parsing telegraf interval rows - err: %v", err)
			dbMu.Unlock()
			return err
		}
		ActiveInterval = append(ActiveInterval, &i)
	}

	ActiveCred = Cred{Id: 0, NetconfUser: "lab", NetconfPwd: "lab123", GnmiUser: "lab", GnmiPwd: "lab123", UseTls: "no", SkipVerify: "yes", ClientTls: "no"}
	rows, err = db.Query("SELECT * FROM credentials;")
	if err != nil {
		logger.Log.Errorf("Error while selecting credentials - err: %v", err)
		dbMu.Unlock()
		return err
	}
	defer rows.Close()
	i := rows.Next()
	if !i {
		// nothing in the DB regarding credential - add default one
		if _, err := db.Exec("INSERT INTO credentials VALUES(?,?,?,?,?,?,?,?);", 0, ActiveCred.NetconfUser, ActiveCred.NetconfPwd, ActiveCred.GnmiUser, ActiveCred.GnmiPwd, ActiveCred.UseTls, ActiveCred.SkipVerify, ActiveCred.ClientTls); err != nil {
			logger.Log.Errorf("Error while adding default credential - err: %v", err)
			dbMu.Unlock()
			return err
		}
	} else {
		err = rows.Scan(&ActiveCred.Id, &ActiveCred.NetconfUser, &ActiveCred.NetconfPwd, &ActiveCred.GnmiUser, &ActiveCred.GnmiPwd, &ActiveCred.UseTls, &ActiveCred.SkipVerify, &ActiveCred.ClientTls)
		if err != nil {
			logger.Log.Errorf("Error while parsing credential rows - err: %v", err)
			dbMu.Unlock()
			return err
		}
	}

	ActiveAdmin = Admin{}
	rows, err = db.Query("SELECT * FROM administration;")
	if err != nil {
		logger.Log.Errorf("Error while selecting administration - err: %v", err)
		dbMu.Unlock()
		return err
	}
	defer rows.Close()
	i = rows.Next()
	if !i {
		// nothing in the DB regarding administration  - add default one
		if _, err := db.Exec("INSERT INTO administration VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?);", 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, influx.DefaultRetention); err != nil {
			logger.Log.Errorf("Error while adding default administration - err: %v", err)
			dbMu.Unlock()
			return err
		}
	} else {
		// Manage new fields
		colExists := false
		rows, err := db.Query("PRAGMA table_info(administration);")
		if err != nil {
			logger.Log.Errorf("Error while checking table info - err: %v", err)
			dbMu.Unlock()
			return err
		}
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				logger.Log.Errorf("Error scanning table_info - err: %v", err)
				dbMu.Unlock()
				return err
			}
			if name == "rpduration" {
				colExists = true
				break
			}
		}
		rows.Close()
		if !colExists {
			_, err := db.Exec("ALTER TABLE administration ADD COLUMN rpduration TEXT DEFAULT '" + influx.DefaultRetention + "';")
			if err != nil {
				logger.Log.Errorf("Error adding rpduration column - err: %v", err)
				dbMu.Unlock()
				return err
			}
		}
		// End of the specific piece of code managing new fields
		rows, err = db.Query("SELECT * FROM administration;")
		if err != nil {
			logger.Log.Errorf("Error while selecting administration - err: %v", err)
			dbMu.Unlock()
			return err
		}
		defer rows.Close()
		rows.Next()
		err = rows.Scan(&ActiveAdmin.Id, &ActiveAdmin.MXDebug, &ActiveAdmin.PTXDebug, &ActiveAdmin.ACXDebug, &ActiveAdmin.EXDebug, &ActiveAdmin.QFXDebug, &ActiveAdmin.SRXDebug, &ActiveAdmin.CRPDDebug, &ActiveAdmin.CPTXDebug, &ActiveAdmin.VMXDebug, &ActiveAdmin.VSRXDebug, &ActiveAdmin.VJUNOSDebug, &ActiveAdmin.VEVODebug, &ActiveAdmin.RPDuration)
		if err != nil {
			logger.Log.Errorf("Error while parsing administration rows - err: %v", err)
			dbMu.Unlock()
			return err
		}
	}

	dbMu.Unlock()
	return nil
}

func CloseDb() error {
	logger.Log.Info("Closing database.")
	return db.Close()
}
