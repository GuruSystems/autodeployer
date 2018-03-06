package main

import (
	_ "github.com/lib/pq"
	"sync"
)

var (
	appidlock sync.Mutex
)

type appdef struct {
	Namespace  string
	Groupname  string
	Repository string
	Appname    string
}

// gets an ID for a given app - returns ID or error
// creates on-the-fly if necessary
func GetOrCreateAppID(namespace, groupname, repository, appname string) (uint64, error) {
	ad := appdef{
		Namespace:  namespace,
		Groupname:  groupname,
		Repository: repository,
		Appname:    appname,
	}
	id, err := GetAppID(&ad)
	if err != nil {
		return 0, err
	}
	if id > 0 {
		return id, nil
	}
	// check-and-create cycle within a lock
	// it happens rarely. (only when a new application is added and logs for
	// the very first time.
	// if necessary, a possible optimisation might be to lock specific to an appdef
	appidlock.Lock()
	defer appidlock.Unlock()
	id, err = GetAppID(&ad)
	if err != nil {
		return 0, err
	}
	if id > 0 {
		return id, nil
	}
	// app does not exist - insert it
	var rid uint64
	err = dbcon.QueryRow("INSERT INTO application (namespace,groupname,repository,appname) values ($1,$2,$3,$4) RETURNING id", ad.Namespace, ad.Groupname, ad.Repository, ad.Appname).Scan(&rid)
	if err != nil {
		return 0, err
	}
	return rid, nil

}

// returns id or 0 if none found
func GetAppID(ad *appdef) (uint64, error) {
	var id uint64
	rows, err := dbcon.Query("select id from application where namespace = $1 and groupname = $2 and repository = $3 and appname = $4", ad.Namespace, ad.Groupname, ad.Repository, ad.Appname)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	return 0, nil
}
