package sqlite

import (
	"database/sql"
	"jtso/logger"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type RtrEntry struct {
	Id        int
	Hostname  string
	Shortname string
	Login     string
	Pwd       string
	Family    string
	Profile   string
}

var db *sql.DB
var dbMu *sync.Mutex
var RtrList []*RtrEntry

func Init(f string) error {
	var err error
	err = nil
	dbMu = new(sync.Mutex)

	db, err = sql.Open("sqlite3", f)
	if err != nil {
		logger.Log.Infof("Error while opening DB %s - err: %v", f, err)
		return err
	}
	const create string = `
		CREATE TABLE IF NOT EXISTS routers (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEST,
		short TEST,
		login TEXT,
		pwd TEXT,
		family TEXT,
		profile TEXT
		);`

	if _, err := db.Exec(create); err != nil {
		logger.Log.Infof("Error while init DB %s - err: %v", f, err)
		return err
	}
	err = LoadAll()
	return err
}

func AddRouter(n string, s string, l string, p string, f string) error {
	dbMu.Lock()
	if _, err := db.Exec("INSERT INTO routers VALUES(NULL,?,?,?,?,?,?);", n, s, l, p, f, ""); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
}

func DelRouter(n string) error {
	dbMu.Lock()
	if _, err := db.Exec("DELETE FROM routers WHERE name=?;", n); err != nil {
		logger.Log.Errorf("Error while adding router %s - err: %v", n, err)
		dbMu.Unlock()
		return err
	}
	dbMu.Unlock()
	err := LoadAll()
	return err
}

func UpdateRouterProfile(n string, p string) error {
	dbMu.Lock()
	if _, err := db.Exec("UPDATE routers SET profile=? WHERE name=?;", p, n); err != nil {
		logger.Log.Errorf("Error while updating router profile %s - err: %v", n, err)
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
		err = rows.Scan(&i.Id, &i.Hostname, &i.Shortname, &i.Login, &i.Pwd, &i.Family, &i.Profile)
		if err != nil {
			logger.Log.Errorf("Error while parsing rows - err: %v", err)
			dbMu.Unlock()
			return err
		}
		RtrList = append(RtrList, &i)
	}
	dbMu.Unlock()
	return nil

}

func CloseDb() error {
	logger.Log.Info("Closing database.")
	return db.Close()
}
