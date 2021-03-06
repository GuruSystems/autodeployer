package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	apb "golang.conradwood.net/autodeployer/proto"
	pb "golang.conradwood.net/deploymonkey/proto"
	"golang.conradwood.net/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"os"
	"strconv"
	"sync"
	"time"
)

// static variables for flag parser
var (
	limit            = flag.Int("limit", 20, "max entries to return when querying for lists")
	port             = flag.Int("port", 4999, "The server port")
	dbhost           = flag.String("dbhost", "postgres", "hostname of the postgres database rdms")
	dbdb             = flag.String("database", "deploymonkey", "database to use for authentication")
	dbuser           = flag.String("dbuser", "root", "username for the database to use for authentication")
	dbpw             = flag.String("dbpw", "pw", "password for the database to use for authentication")
	file             = flag.String("filename", "", "filename with a group definition (for testing)")
	applyonly        = flag.Bool("apply_only", false, "if true will apply current config and exit")
	testmode         = flag.Bool("testmode", false, "sets some stuff to make it more convenient to test")
	reapply_on_start = flag.Bool("reapply", false, "set to true if you want deploymonkey to reset all versions (shuts down all services and restarts them!!")
	applyinterval    = flag.Int("apply_interval", 60, "`seconds` between scans for discrepancies and re-applying them")
	list             = flag.String("list", "", "list this `repository` previous versions")
	dbcon            *sql.DB
	dbinfo           string
	applyLock        sync.Mutex
	curApply         *applyingInfo
)

type applyingInfo struct {
	version int
}

type appVersionDef struct {
	appdef *pb.ApplicationDefinition
	gv     *groupVersion
}
type groupVersion struct {
	Version int
	GroupID int
	Created time.Time
}

// callback from the compound initialisation
func st(server *grpc.Server) error {
	s := new(DeployMonkey)
	// Register the handler object
	pb.RegisterDeployMonkeyServer(server, s)
	return nil
}

func main() {
	var err error
	flag.Parse() // parse stuff. see "var" section above

	err = initDB()
	if err != nil {
		fmt.Printf("Failed to initdb(): %s\n", err)
		dbcon = nil
	}
	if *list != "" {
		listVersions(*list)
		os.Exit(0)
	}

	if *file != "" {
		importFile(*file)
		if dbcon != nil {
			dbcon.Close()
		}
		os.Exit(0)
	}
	if (!*testmode) && (*reapply_on_start) {
		err := applyAllVersions(false)
		if err != nil {
			fmt.Printf("Failed to apply all versions: %s\n", err)
		}
	}
	if *applyonly {
		if dbcon != nil {
			dbcon.Close()
		}
		os.Exit(10)
	}
	if !*testmode {
		applyTimer := time.NewTimer(time.Second * time.Duration(*applyinterval))
		go func() {
			<-applyTimer.C
			applyAllVersions(true)
		}()
	}
	sd := server.NewServerDef()
	sd.Port = *port
	sd.Register = st
	err = server.ServerStartup(sd)
	if err != nil {
		fmt.Printf("failed to start server: %s\n", err)
	}
	if dbcon != nil {
		dbcon.Close()
	}
	fmt.Printf("Done\n")
	return
}

func importFile(filename string) {
	fd, err := ParseFile(filename)
	if err != nil {
		fmt.Printf("Failed to parse file %s: %s\n", filename, err)
		return
	}
	dm := new(DeployMonkey)
	for _, gdr := range fd.Groups {
		fmt.Printf("Group: %s\n", gdr.GroupID)
		gresp, err := dm.DefineGroup(nil, gdr)
		if err != nil {
			fmt.Printf("Failed to manage group %s: %s\n", gdr.GroupID, err)
			return
		}
		fmt.Printf("Result: %s\n", gresp.Result)
		if gresp.Result == pb.GroupResponseStatus_NOCHANGE {
			fmt.Printf("Aborted\n")
			return
		}
		fmt.Printf("Deploying:\n")
		dr := pb.DeployRequest{VersionID: gresp.VersionID}
		_, err = dm.DeployVersion(nil, &dr)
		if err != nil {
			fmt.Printf("Failed to deploy group %s (version %d): %s\n", gdr.GroupID, gresp.VersionID, err)
			return
		}
	}
	fmt.Printf("File parsed.\n")

}

/**********************************
* catch-all fix up
***********************************/
// this gets all groups, all current versions
// and makes the deployment match
// if pendingonly is true, will check for mismatched current != pending versions
// and only apply those
func applyAllVersions(pendingonly bool) error {
	var dv int
	if pendingonly {
		fmt.Printf("(Re)applying all pending versions...\n")
	} else {
		fmt.Printf("Reapplying all current versions...\n")
	}
	err := initDB()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to initdb: %s", err))
	}
	var rows *sql.Rows
	if pendingonly {
		rows, err = dbcon.Query("SELECT pendingversion from appgroup where deployedversion != pendingversion")
	} else {
		rows, err = dbcon.Query("SELECT deployedversion from appgroup ")
	}
	if err != nil {
		fmt.Printf("Failed to get deployedversions: %s\n", err)
		return err
	}
	for rows.Next() {
		err := rows.Scan(&dv)
		if err != nil {
			fmt.Printf("Failed to get deployedversion for a group: %s\n", err)
			return err
		}
		err = applyVersion(dv)
		if err != nil {
			fmt.Printf("Failed to apply version %d: %s\n", dv, err)
		}
	}
	return nil
}

/**********************************
* implementing the postgres functions here:
***********************************/
func initDB() error {
	var err error
	var now string
	if dbcon != nil {
		return nil
	}
	host := *dbhost
	username := *dbuser
	database := *dbdb
	password := *dbpw
	fmt.Printf("Connecting to host %s\n", host)

	dbinfo = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=require",
		host, username, password, database)
	dbcon, err = sql.Open("postgres", dbinfo)
	if err != nil {
		fmt.Printf("Failed to connect to %s on host \"%s\" as \"%s\"\n", database, host, username)
		return err
	}
	err = dbcon.QueryRow("SELECT NOW() as now").Scan(&now)
	if err != nil {
		fmt.Printf("Failed to scan %s on host \"%s\" as \"%s\"\n", database, host, username)
		return err
	}
	fmt.Printf("Time in database: %s\n", now)
	return nil
}

type DBGroup struct {
	id              int
	DeployedVersion int
	PendingVersion  int
	groupDef        *pb.GroupDefinitionRequest
}

// get the group with given name from database. if no such group will return nil
func getGroupFromDatabase(nameSpace string, groupName string) (*DBGroup, error) {
	res := pb.GroupDefinitionRequest{}
	d := DBGroup{}
	res.Namespace = nameSpace
	rows, err := dbcon.Query("SELECT id,groupname,deployedversion,pendingversion from appgroup where groupname=$1 and namespace=$2", groupName, nameSpace)
	if err != nil {
		fmt.Printf("Failed to get groupname %s\n", groupName)
		return nil, err
	}
	gotone := false
	for rows.Next() {
		gotone = true
		err := rows.Scan(&d.id, &res.GroupID, &d.DeployedVersion, &d.PendingVersion)
		if err != nil {
			fmt.Printf("Failed to get row for groupname %s\n", groupName)
			return nil, err
		}
	}
	if !gotone {
		return nil, nil
	}
	d.groupDef = &res
	return &d, nil

}
func createGroup(nameSpace string, groupName string) (*DBGroup, error) {
	_, err := dbcon.Exec("INSERT into appgroup (groupname,namespace) values ($1,$2)", groupName, nameSpace)
	if err != nil {
		return nil, err
	}
	return getGroupFromDatabase(nameSpace, groupName)

}

// create a new group version, return versionID
func createGroupVersion(nameSpace string, groupName string, def []*pb.ApplicationDefinition) (string, error) {
	var id int
	r, err := getGroupFromDatabase(nameSpace, groupName)
	if err != nil {
		return "", err
	}
	if r.groupDef.GroupID == "" {
		// had no row!
		r, err = createGroup(nameSpace, groupName)
		if err != nil {
			return "", err
		}
	}
	err = dbcon.QueryRow("INSERT into group_version (group_id) values ($1) RETURNING id", r.id).Scan(&id)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to insert group_version: %s", err))
	}
	versionId := id
	fmt.Printf("New Version: %d for Group #%d\n", versionId, r.id)
	for _, ad := range def {
		fmt.Printf("Saving: %v\n", ad)
		id, err := saveApp(ad)
		if err != nil {
			return "", err
		}
		fmt.Printf("Inserted App #%s\n", id)
		_, err = dbcon.Exec("INSERT into lnk_app_grp (group_version_id,app_id) values ($1,$2)", versionId, id)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Failed to add application to new version: %s", err))
		}
	}
	return fmt.Sprintf("%d", versionId), nil
}

func saveApp(app *pb.ApplicationDefinition) (string, error) {
	var id int
	err := CheckAppComplete(app)
	if err != nil {
		return "", err
	}
	err = dbcon.QueryRow("INSERT into appdef (sourceurl,downloaduser,downloadpw,executable,repo,buildid,instances,mgroup,deploytype) values ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id",
		app.DownloadURL, app.DownloadUser, app.DownloadPassword,
		app.Binary, app.Repository, app.BuildID, app.Instances, app.Machines, app.DeployType).Scan(&id)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to insert application: %s", err))
	}
	for _, arg := range app.Args {
		_, err = dbcon.Exec("INSERT INTO args (argument,app_id) values ($1,$2)", arg, id)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Failed to insert tag for app %d: %s", id, err))
		}
	}
	for _, ar := range app.AutoRegs {
		_, err = dbcon.Exec("INSERT INTO autoreg (portdef,servicename,apitypes,app_id) values ($1,$2,$3,$4)", ar.Portdef, ar.ServiceName, ar.ApiTypes, id)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Failed to insert autoreg for app %d: %s", id, err))
		}
	}
	return fmt.Sprintf("%d", id), nil
}

// given a group version will load all its apps into objects
func getGroupLatestVersion(namespace string, groupname string) (int, error) {
	rows, err := dbcon.Query("SELECT MAX(group_version.id) as maxid from group_version,appgroup where group_id = appgroup.id and appgroup.namespace = $1 and appgroup.groupname = $2", namespace, groupname)
	if err != nil {
		fmt.Printf("Failed to get latest version for (%s,%s):%s\n", namespace, groupname, err)
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var maxid int
		err = rows.Scan(&maxid)
		if err != nil {
			fmt.Printf("Failed to scan for latest version for (%s,%s):%s\n", namespace, groupname, err)
			return 0, err
		}
		return maxid, nil
	}
	return 0, nil
}

// given a group version will load all its apps into objects
func loadAppGroupVersion(version int) ([]*pb.ApplicationDefinition, error) {
	var res []*pb.ApplicationDefinition
	if *testmode {
		fmt.Printf("Loading appgroup version #%d\n", version)
	}
	rows, err := dbcon.Query("SELECT appdef.id,sourceurl,downloaduser,downloadpw,executable,repo,buildid,instances,mgroup,deploytype from appdef, lnk_app_grp where appdef.id = lnk_app_grp.app_id and lnk_app_grp.group_version_id = $1", version)
	if err != nil {
		fmt.Printf("Failed to get apps for version %d:%s\n", version, err)
		return nil, err
	}
	for rows.Next() {
		r, err := loadApp(rows)
		if err != nil {
			fmt.Printf("Failed to get app for version %d:%s\n", version, err)
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// turns a database row into an applicationdefinition object
// optionally suppling an interface to take up additional values
func loadApp(row *sql.Rows, dest ...interface{}) (*pb.ApplicationDefinition, error) {
	var id int
	res := pb.ApplicationDefinition{}
	var t []interface{}
	t = append(t, &id, &res.DownloadURL, &res.DownloadUser, &res.DownloadPassword,
		&res.Binary, &res.Repository, &res.BuildID, &res.Instances, &res.Machines, &res.DeployType)
	for _, z := range dest {
		t = append(t, z)
	}
	err := row.Scan(t...)
	if err != nil {
		return nil, err
	}
	args, err := loadAppArgs(id)
	if err != nil {
		return nil, err
	}
	res.Args = args

	regs, err := loadAutoReg(id)
	if err != nil {
		return nil, err
	}
	res.AutoRegs = regs
	return &res, nil
}

// given an application, it loads the args from DB
func loadAppArgs(id int) ([]string, error) {
	var res []string
	var s string
	rows, err := dbcon.Query("SELECT argument from args where app_id = $1", id)
	if err != nil {
		s := fmt.Sprintf("Failed to get tags for app %d:%s\n", id, err)
		return nil, errors.New(s)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&s)
		if err != nil {
			s := fmt.Sprintf("Failed to get tag for app %d:%s\n", id, err)
			return nil, errors.New(s)
		}
		res = append(res, s)
	}
	return res, nil
}

// given an applicationid, it loads the args from DB
func loadAutoReg(id int) ([]*apb.AutoRegistration, error) {
	var res []*apb.AutoRegistration
	if *testmode {
		fmt.Printf("Loading auto registration for app_id %d\n", id)
	}
	rows, err := dbcon.Query("SELECT portdef,servicename,apitypes from autoreg where app_id = $1", id)
	if err != nil {
		s := fmt.Sprintf("Failed to get autoregs for app %d:%s\n", id, err)
		return nil, errors.New(s)
	}
	defer rows.Close()
	for rows.Next() {
		ar := &apb.AutoRegistration{}
		err = rows.Scan(&ar.Portdef, &ar.ServiceName, &ar.ApiTypes)
		if err != nil {
			s := fmt.Sprintf("Failed to get autoreg for app %d:%s\n", id, err)
			return nil, errors.New(s)
		}
		res = append(res, ar)
	}
	return res, nil
}

// get group id from version
func getGroupIDFromVersion(v int) (*groupVersion, error) {
	gr := groupVersion{Version: v}
	err := dbcon.QueryRow("select group_id,created from group_version where id = $1", v).Scan(&gr.GroupID, &gr.Created)
	if err != nil {
		return nil, err
	}
	return &gr, nil
}

// update the deployed version of a group (group referred to by version!)
func updateDeployedVersionNumber(v int) error {
	gid, err := getGroupIDFromVersion(v)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid Group-Version: \"%d\": %s", v, err))
	}
	_, err = dbcon.Exec("update appgroup set deployedversion = $1 where id = $2", v, gid.GroupID)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to update group: %s", err))
	}
	fmt.Printf("Updated deployedversion to %d\n", v)
	return nil
}

// given a version of a group checks the workers and fixes it up to match version
func applyVersion(v int) error {
	fmt.Printf("Waiting for applyVersionLock() (held by %v)...\n", curApply)
	applyLock.Lock()
	defer applyLock.Unlock()
	curApply = &applyingInfo{
		version: v,
	}
	fmt.Printf("Applying version %d\n", v)
	// first step: mark the version as pending
	// so if it fails for some reason, we know what to replay
	gid, err := getGroupIDFromVersion(v)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid Group-Version: \"%d\": %s", v, err))
	}
	_, err = dbcon.Exec("update appgroup set pendingversion = $1 where id = $2", v, gid.GroupID)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to update group: %s", err))
	}
	var ns, gn string
	err = dbcon.QueryRow("SELECT namespace,groupname from appgroup where id = $1", gid.GroupID).Scan(&ns, &gn)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to get groupnames: %s", err))
	}
	dbgroup, err := getGroupFromDatabase(ns, gn)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to get group (%s,%s) from db: %s", ns, gn, err))
	}
	apps, err := loadAppGroupVersion(v)
	if err != nil {
		return errors.New(fmt.Sprintf("error loading apps for version %d: %s", v, err))
	}
	fmt.Printf("Makeitso (%d)...\n", v)
	err = MakeItSo(dbgroup, apps)
	if err != nil {
		return errors.New(fmt.Sprintf("error applyings apps for version %d: %s", v, err))
	}
	NotifyPeopleAboutDeploy(dbgroup, apps, v)
	return nil
}

/**********************************
* implementing the server functions here:
***********************************/
type DeployMonkey struct{}

func (s *DeployMonkey) DefineGroup(ctx context.Context, cr *pb.GroupDefinitionRequest) (*pb.GroupDefResponse, error) {
	if cr.Namespace == "" {
		return nil, errors.New("Namespace required")
	}
	if cr.GroupID == "" {
		return nil, errors.New("GroupID required")
	}
	err := initDB()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to initdb: %s", err))
	}
	cur, err := getGroupFromDatabase(cr.Namespace, cr.GroupID)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get group from db: %s", err))
	}
	if cur == nil {
		cur, err = createGroup(cr.Namespace, cr.GroupID)
		if err != nil {
			return nil, err
		}
	}
	apps, err := loadAppGroupVersion(cur.DeployedVersion)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get apps for version %d from db: %s", cur.DeployedVersion, err))
	}
	cur.groupDef.Applications = apps
	fmt.Printf("Loaded Group from database: \n")
	PrintGroup(cur.groupDef)
	diff, err := Compare(cur.groupDef, cr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to compare: %s", err))
	}
	if len(diff.AppDiffs) == 0 {
		r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_NOCHANGE}
		return &r, nil
	}

	// in diff we now have a list of appdiffs (stuff we need to change)
	for _, dg := range diff.AppDiffs {
		fmt.Printf("Update: %s\n", dg.Describe())
	}

	// create a new version with our new app definitions
	vid, err := createGroupVersion(cr.Namespace, cr.GroupID, cr.Applications)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create new version: %s", err))
	}

	// tell client we have saved the changes (return an ID to refer to this version)
	// (to call DeployVersion() later)
	r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_CHANGEACCEPTED,
		VersionID: vid,
	}
	return &r, nil
}

// given a Version# -> Take it online ("Make it so")
func (s *DeployMonkey) DeployVersion(ctx context.Context, cr *pb.DeployRequest) (*pb.DeployResponse, error) {
	if cr.VersionID == "" {
		return nil, errors.New("VersionID required for deployment")
	}
	version, err := strconv.Atoi(cr.VersionID)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Invalid VersionID: \"%s\": %s", cr.VersionID, err))
	}
	err = applyVersion(version)
	if err != nil {
		return nil, err
	}
	updateDeployedVersionNumber(version)
	r := pb.DeployResponse{}
	return &r, nil
}

// updates all apps in a repo to a new buildid
func (s *DeployMonkey) UpdateRepo(ctx context.Context, cr *pb.UpdateRepoRequest) (*pb.GroupDefResponse, error) {
	if cr.Namespace == "" {
		return nil, errors.New("Namespace required")
	}
	if cr.GroupID == "" {
		return nil, errors.New("GroupID required")
	}
	if cr.Repository == "" {
		return nil, errors.New("App Repository required")
	}
	fmt.Printf("Updating all apps in repository %s in (%s,%s) to buildid: %d\n", cr.Repository,
		cr.Namespace, cr.GroupID, cr.BuildID)
	err := initDB()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to initdb: %s", err))
	}
	cur, err := getGroupFromDatabase(cr.Namespace, cr.GroupID)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get group from db: %s", err))
	}
	if cur == nil {
		return nil, errors.New(fmt.Sprintf("No such group: (%s,%s)", cr.Namespace, cr.GroupID))
	}

	lastVersion, err := getGroupLatestVersion(cr.Namespace, cr.GroupID)
	if err != nil {
		return nil, err
	}
	apps, err := loadAppGroupVersion(lastVersion)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get apps for version %d from db: %s", cur.DeployedVersion, err))
	}
	fmt.Printf("Loaded Group from database: \n")
	cur.groupDef.Applications = apps
	PrintGroup(cur.groupDef)
	// now find the app we want to update:
	foundone := false
	for _, app := range apps {
		if app.Repository != cr.Repository {
			continue
		}
		fmt.Printf("Updating app: %s\n", app.Repository)
		app.BuildID = cr.BuildID
		foundone = true
	}
	if !foundone {
		fmt.Printf("Nothing to update, app is already up to date\n")
		r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_NOCHANGE}
		return &r, nil
	}
	cur.groupDef.Applications = apps
	fmt.Printf("Updated Group: \n")
	cur.groupDef.Applications = apps
	PrintGroup(cur.groupDef)

	sv, err := createGroupVersion(cr.Namespace, cr.GroupID, apps)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create a new group version: %s", err))
	}
	fmt.Printf("Created group version: %s\n", sv)
	version, err := strconv.Atoi(sv)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("version group not a number (%s)? BUG!: %s", sv, err))
	}
	err = applyVersion(version)
	if err != nil {
		return nil, err
	}
	updateDeployedVersionNumber(version)
	r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_CHANGEACCEPTED}
	return &r, nil

}

// updates a single app to a new version
func (s *DeployMonkey) UpdateApp(ctx context.Context, cr *pb.UpdateAppRequest) (*pb.GroupDefResponse, error) {
	if cr.App.BuildID == 0 {
		return nil, errors.New("BuildID 0 is invalid")
	}
	if cr.Namespace == "" {
		return nil, errors.New("Namespace required")
	}
	if cr.GroupID == "" {
		return nil, errors.New("GroupID required")
	}
	if cr.App.Repository == "" {
		return nil, errors.New("App Repository required")
	}
	fmt.Printf("Request to update app:\n")
	PrintApp(cr.App)
	err := initDB()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to initdb: %s", err))
	}
	cur, err := getGroupFromDatabase(cr.Namespace, cr.GroupID)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get group from db: %s", err))
	}
	if cur == nil {
		return nil, errors.New(fmt.Sprintf("No such group: (%s,%s)", cr.Namespace, cr.GroupID))
	}

	lastVersion, err := getGroupLatestVersion(cr.Namespace, cr.GroupID)
	if err != nil {
		return nil, err
	}
	apps, err := loadAppGroupVersion(lastVersion)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get apps for version %d from db: %s", cur.DeployedVersion, err))
	}
	fmt.Printf("Loaded Group from database: \n")
	cur.groupDef.Applications = apps
	PrintGroup(cur.groupDef)
	// now find the app we want to update:
	foundone := false
	for _, app := range apps {
		if isSame(app, cr.App) {
			fmt.Printf("Updating app: %s\n", app.Repository)
			m := mergeApp(app, cr.App)
			if !m {
				fmt.Printf("Nothing to update, app is already up to date\n")
				r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_NOCHANGE}
				return &r, nil
			}
			foundone = true
			break
		}
	}
	if !foundone {
		return nil, errors.New(fmt.Sprintf("There is no app \"%s\" in group (%s,%s)", cr.App.Repository, cr.Namespace, cr.GroupID))
	}
	cur.groupDef.Applications = apps
	fmt.Printf("Updated Group: \n")
	cur.groupDef.Applications = apps
	PrintGroup(cur.groupDef)

	sv, err := createGroupVersion(cr.Namespace, cr.GroupID, apps)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create a new group version: %s", err))
	}
	fmt.Printf("Created group version: %s\n", sv)
	version, err := strconv.Atoi(sv)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("version group not a number (%s)? BUG!: %s", sv, err))
	}
	err = applyVersion(version)
	if err != nil {
		return nil, err
	}
	updateDeployedVersionNumber(version)
	r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_CHANGEACCEPTED}
	return &r, nil
}

// merge source into target
// basically anything set in source shall be copied to target
// returns true if there was something updated
func mergeApp(t, s *pb.ApplicationDefinition) bool {
	res := false
	if (s.DownloadURL != "") && (s.DownloadURL != t.DownloadURL) {
		res = true
		t.DownloadURL = s.DownloadURL
	}
	if (s.DownloadUser != "") && (s.DownloadUser != t.DownloadUser) {
		res = true
		t.DownloadUser = s.DownloadUser
	}
	if (s.DownloadPassword != "") && (s.DownloadPassword != t.DownloadPassword) {
		res = true
		t.DownloadPassword = s.DownloadPassword
	}
	if (len(s.Args) != 0) && (!AreArgsIdentical(s, t)) {
		res = true
		t.Args = s.Args
	}
	if (s.Binary != "") && (s.Binary != t.Binary) {
		res = true
		t.Binary = s.Binary
	}
	if (s.BuildID != 0) && (s.BuildID != t.BuildID) {
		res = true
		t.BuildID = s.BuildID
	}
	if (s.Instances != 0) && (s.Instances != t.Instances) {
		res = true
		t.Instances = s.Instances
	}
	return res
}

func (s *DeployMonkey) GetNameSpaces(ctx context.Context, cr *pb.GetNameSpaceRequest) (*pb.GetNameSpaceResponse, error) {
	resp := pb.GetNameSpaceResponse{}
	n, err := getStringsFromDB("select distinct namespace from appgroup", "")
	if err != nil {
		return nil, err
	}
	resp.NameSpaces = n
	return &resp, nil
}

func (s *DeployMonkey) GetGroups(ctx context.Context, cr *pb.GetGroupsRequest) (*pb.GetGroupsResponse, error) {
	if cr.NameSpace == "" {
		return nil, errors.New("Namespace required")
	}
	resp := pb.GetGroupsResponse{}
	n, err := getStringsFromDB("select groupname from appgroup where namespace = $1", cr.NameSpace)
	if err != nil {
		return nil, err
	}
	for _, name := range n {
		dbg, err := getGroupFromDatabase(cr.NameSpace, name)
		if err != nil {
			return nil, err
		}
		gd := pb.GroupDef{DeployedVersion: int64(dbg.DeployedVersion),
			PendingVersion: int64(dbg.PendingVersion),
			GroupID:        dbg.groupDef.GroupID,
			NameSpace:      dbg.groupDef.Namespace,
		}
		resp.Groups = append(resp.Groups, &gd)

	}

	return &resp, nil
}
func (s *DeployMonkey) ListVersionsForGroup(ctx context.Context, cr *pb.ListVersionRequest) (*pb.GetAppVersionsResponse, error) {
	avds, err := listVersions(cr.Repository)
	if err != nil {
		return nil, err
	}
	res := pb.GetAppVersionsResponse{}
	for _, avd := range avds {
		gar := pb.GetAppResponse{
			Created:     avd.gv.Created.Unix(),
			VersionID:   int64(avd.gv.Version),
			Application: avd.appdef,
		}
		res.Apps = append(res.Apps, &gar)
	}
	return &res, nil
}
func listVersions(repo string) ([]*appVersionDef, error) {
	if dbcon == nil {
		return nil, errors.New("database not open")
	}
	// this query gives us the version in lnk_app_grp.group_version_id
	rows, err := dbcon.Query("SELECT appdef.id,sourceurl,downloaduser,downloadpw,executable,repo,buildid,instances,mgroup,deploytype,lnk_app_grp.group_version_id,group_version.created from appdef, lnk_app_grp,group_version where appdef.id = lnk_app_grp.app_id and group_version.id = lnk_app_grp.group_version_id and repo = $1 order by group_version.id desc limit $2", repo, *limit) //and lnk_app_grp.group_version_id = $1", version)
	if err != nil {
		fmt.Printf("Failed to get apps:%s\n", err)
		return nil, err
	}
	var res []*appVersionDef
	for rows.Next() {
		gv := groupVersion{}
		ad, err := loadApp(rows, &gv.Version, &gv.Created)
		if err != nil {
			fmt.Printf("Failed to get apps:%s\n", err)
			return nil, err
		}
		fmt.Printf("Version #%d: BuildID: %d, repository=%s (%v)\n", gv.Version, ad.BuildID, ad.Repository, gv.Created)
		r := appVersionDef{
			appdef: ad,
			gv:     &gv,
		}
		res = append(res, &r)
	}
	return res, nil
}

func getStringsFromDB(sqls string, val string) ([]string, error) {
	err := initDB()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to initdb: %s", err))
	}
	var rows *sql.Rows
	if val != "" {
		rows, err = dbcon.Query(sqls, val)
	} else {
		rows, err = dbcon.Query(sqls)
	}
	if err != nil {
		fmt.Printf("Failed to query \"%s\": %s\n", sqls, err)
		return nil, err
	}
	var res []string
	var dv string
	for rows.Next() {
		err := rows.Scan(&dv)
		if err != nil {
			fmt.Printf("Failed to get deployedversion for a group: %s\n", err)
			return nil, err
		}
		res = append(res, dv)
	}
	return res, nil
}

func (s *DeployMonkey) GetApplications(ctx context.Context, cr *pb.GetAppsRequest) (*pb.GetAppsResponse, error) {
	dbg, err := getGroupFromDatabase(cr.NameSpace, cr.GroupName)
	if err != nil {
		s := fmt.Sprintf("No such group: (%s,%s)\n", cr.NameSpace, cr.GroupName)
		fmt.Println(s)
		return nil, errors.New(s)
	}
	fmt.Printf("Listing Deployed Version: %d\n", dbg.DeployedVersion)
	ad, err := loadAppGroupVersion(dbg.DeployedVersion)
	if err != nil {
		s := fmt.Sprintf("No applications for version %d: (%s,%s)\n", dbg.DeployedVersion, cr.NameSpace, cr.GroupName)
		fmt.Println(s)
		return nil, errors.New(s)
	}
	resp := pb.GetAppsResponse{}
	resp.Applications = ad
	return &resp, nil
}

func (s *DeployMonkey) ParseConfigFile(ctx context.Context, cr *pb.ParseRequest) (*pb.ParseConfigResponse, error) {
	fd, err := ParseConfig([]byte(cr.Config))
	if err != nil {
		return nil, err
	}
	res := pb.ParseConfigResponse{}
	res.GroupDef = fd.Groups
	return &res, nil
}
func (s *DeployMonkey) ApplyVersions(ctx context.Context, cr *pb.ApplyRequest) (*pb.EmptyMessage, error) {
	applyAllVersions(!cr.All)
	return &pb.EmptyMessage{}, nil
}
