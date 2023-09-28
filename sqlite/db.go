package sqlite

import (
	"database/sql"
	"jtso/logger"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type RtrEntry struct {
	Id        int
	Hostname  string
	Shortname string
	Family    string
	Profile   int
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
}

var db *sql.DB
var dbMu *sync.Mutex
var RtrList []*RtrEntry
var AssoList []*AssoEntry
var ActiveCred Cred

func Init(f string) error {
	var err error
	err = nil
	dbMu = new(sync.Mutex)

	db, err = sql.Open("sqlite3", f)
	if err != nil {
		logger.Log.Infof("Error while opening DB %s - err: %v", f, err)
		return err
	}
	const createRtr string = `
		CREATE TABLE IF NOT EXISTS routers (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT,
		short TEXT,
		family TEXT,
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
		usetls TEXT
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

func AddRouter(n string, s string, f string) error {
	dbMu.Lock()
	if _, err := db.Exec("INSERT INTO routers VALUES(NULL,?,?,?,?);", n, s, f, 0); err != nil {
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
	err = LoadAll()
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
	err = LoadAll()
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

func UpdateCredentials(nu string, np string, gu string, gp string, t string) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE credentials SET netuser=?, netpwd=?, gnmiuser=?, gnmipwd=?, usetls=?   WHERE id=0;", nu, np, gu, gp, t); err != nil {
		logger.Log.Errorf("Error while updating credential - err: %v", err)
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
		err = rows.Scan(&i.Id, &i.Hostname, &i.Shortname, &i.Family, &i.Profile)
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
		i.Assos = strings.Split(tmpList, "|")
		AssoList = append(AssoList, &i)
	}

	ActiveCred = Cred{Id: 0, NetconfUser: "lab", NetconfPwd: "lab123", GnmiUser: "lab", GnmiPwd: "lab123", UseTls: "no"}
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
		if _, err := db.Exec("INSERT INTO credentials VALUES(?,?,?,?,?,?);", 0, ActiveCred.NetconfUser, ActiveCred.NetconfPwd, ActiveCred.GnmiUser, ActiveCred.GnmiPwd, ActiveCred.UseTls); err != nil {
			logger.Log.Errorf("Error while adding default credential - err: %v", err)
			dbMu.Unlock()
			return err
		}
	} else {
		err = rows.Scan(&ActiveCred.Id, &ActiveCred.NetconfUser, &ActiveCred.NetconfPwd, &ActiveCred.GnmiUser, &ActiveCred.GnmiPwd, &ActiveCred.UseTls)
		if err != nil {
			logger.Log.Errorf("Error while parsing credential rows - err: %v", err)
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
