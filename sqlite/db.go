package sqlite

import (
	"database/sql"
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
}

var db *sql.DB
var dbMu *sync.Mutex
var RtrList []*RtrEntry
var AssoList []*AssoEntry
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
		vevodebug INTEGER
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

func AddRouter(n string, s string, f string, m string, v string) error {
	dbMu.Lock()

	if _, err := db.Exec("INSERT INTO routers VALUES(NULL,?,?,?,?,?,?);", n, s, f, m, v, 0); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
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
	err := LoadAll()
	return err
}

func updateRouterProfile(n string, p int) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE routers SET profile=? WHERE short=?;", p, n); err != nil {
		logger.Log.Errorf("Error while updating router profile %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
}

func UpdateRouter(s string, f string, m string, v string) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE routers SET family=?, model=?, version=? WHERE short=?", f, m, v, s); err != nil {
		logger.Log.Errorf("Error while updating router - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
}

func UpdateCredentials(nu string, np string, gu string, gp string, t string, s string, c string) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE credentials SET netuser=?, netpwd=?, gnmiuser=?, gnmipwd=?, usetls=?, skipverify=?, clienttls=?  WHERE id=0;", nu, np, gu, gp, t, s, c); err != nil {
		logger.Log.Errorf("Error while updating credential - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
}

func UpdateDebugMode(instance string, debug int) error {
	dbMu.Lock()
	// Save debug state
	debugInst := instance + "debug"

	// update the debug value for the instance
	if _, err := db.Exec("UPDATE administration SET "+debugInst+"=? WHERE id=0;", debug); err != nil {
		logger.Log.Errorf("Error while updating administration - err: %v", err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
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
		if _, err := db.Exec("INSERT INTO administration VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?);", 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0); err != nil {
			logger.Log.Errorf("Error while adding default administration - err: %v", err)
			dbMu.Unlock()
			return err
		}
	} else {
		err = rows.Scan(&ActiveAdmin.Id, &ActiveAdmin.MXDebug, &ActiveAdmin.PTXDebug, &ActiveAdmin.ACXDebug, &ActiveAdmin.EXDebug, &ActiveAdmin.QFXDebug, &ActiveAdmin.SRXDebug, &ActiveAdmin.CRPDDebug, &ActiveAdmin.CPTXDebug, &ActiveAdmin.VMXDebug, &ActiveAdmin.VSRXDebug, &ActiveAdmin.VJUNOSDebug, &ActiveAdmin.VEVODebug)
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
