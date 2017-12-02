package main

import (
	"database/sql"
	_ "github.com/lib/pq"

	"errors"
	"flag"
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
	"golang.conradwood.net/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"os"
	"strconv"
)

const (
	DEPLOY_PREFIX = "DM-APPDEF-"
)

// static variables for flag parser
var (
	port      = flag.Int("port", 4999, "The server port")
	dbhost    = flag.String("dbhost", "postgres", "hostname of the postgres database rdms")
	dbdb      = flag.String("database", "deploymonkey", "database to use for authentication")
	dbuser    = flag.String("dbuser", "root", "username for the database to use for authentication")
	dbpw      = flag.String("dbpw", "pw", "password for the database to use for authentication")
	file      = flag.String("filename", "", "filename with a group definition (for testing)")
	applyonly = flag.Bool("apply_only", false, "if true will apply current config and exit")
	dbcon     *sql.DB
	dbinfo    string
)

// callback from the compound initialisation
func st(server *grpc.Server) error {
	s := new(DeployMonkey)
	// Register the handler object
	pb.RegisterDeployMonkeyServer(server, s)
	return nil
}

func main() {
	flag.Parse() // parse stuff. see "var" section above
	if *file != "" {
		importFile(*file)
		if dbcon != nil {
			dbcon.Close()
		}
		os.Exit(0)
	}
	sd := server.ServerDef{
		Port: *port,
	}
	err := applyAllVersions()
	if err != nil {
		fmt.Printf("Failed to apply all versions: %s\n", err)
	}
	if *applyonly {
		if err == nil {
			os.Exit(0)
		}
		os.Exit(10)
	}
	sd.Register = st
	err = server.ServerStartup(sd)
	if err != nil {
		fmt.Printf("failed to start server: %s\n", err)
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
func applyAllVersions() error {
	var dv int
	err := initDB()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to initdb: %s", err))
	}
	rows, err := dbcon.Query("SELECT deployedversion from appgroup ")
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
	groupDef        *pb.GroupDefinitionRequest
}

// get the group with given name from database. if no such group will return nil
func getGroupFromDatabase(nameSpace string, groupName string) (*DBGroup, error) {
	res := pb.GroupDefinitionRequest{}
	d := DBGroup{}
	res.Namespace = nameSpace
	rows, err := dbcon.Query("SELECT id,groupname,deployedversion from appgroup where groupname=$1 and namespace=$2", groupName, nameSpace)
	if err != nil {
		fmt.Printf("Failed to get groupname %s\n", groupName)
		return nil, err
	}
	gotone := false
	for rows.Next() {
		gotone = true
		err := rows.Scan(&d.id, &res.GroupID, &d.DeployedVersion)
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
	err = dbcon.QueryRow("INSERT into appdef (sourceurl,downloaduser,downloadpw,executable,repo,buildid,instances) values ($1,$2,$3,$4,$5,$6,$7) RETURNING id",
		app.DownloadURL, app.DownloadUser, app.DownloadPassword,
		app.Binary, app.Repository, app.BuildID, app.Instances).Scan(&id)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to insert application: %s", err))
	}
	for _, arg := range app.Args {
		_, err = dbcon.Exec("INSERT INTO args (argument,app_id) values ($1,$2)", arg, id)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Failed to insert tag for app %d: %s", id, err))
		}
	}
	return fmt.Sprintf("%d", id), nil
}

// given a group version will load all its apps into objects
func loadAppGroupVersion(version int) ([]*pb.ApplicationDefinition, error) {
	var res []*pb.ApplicationDefinition
	rows, err := dbcon.Query("SELECT appdef.id,sourceurl,downloaduser,downloadpw,executable,repo,buildid,instances from appdef, lnk_app_grp where appdef.id = lnk_app_grp.app_id and lnk_app_grp.group_version_id = $1", version)
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
		r.DeploymentID = fmt.Sprintf("%s%d", DEPLOY_PREFIX, version)
		res = append(res, r)
	}
	return res, nil
}

// turns a database row into an applicationdefinition object
func loadApp(row *sql.Rows) (*pb.ApplicationDefinition, error) {
	var id int
	res := pb.ApplicationDefinition{}
	err := row.Scan(&id, &res.DownloadURL, &res.DownloadUser, &res.DownloadPassword,
		&res.Binary, &res.Repository, &res.BuildID, &res.Instances)
	if err != nil {
		return nil, err
	}
	args, err := loadAppArgs(id)
	if err != nil {
		return nil, err
	}
	res.Args = args
	return &res, nil
}

// turns a database row into an applicationdefinition object
func loadAppArgs(id int) ([]string, error) {
	var res []string
	var s string
	rows, err := dbcon.Query("SELECT argument from args where app_id = $1", id)
	if err != nil {
		s := fmt.Sprintf("Failed to get tags for app %d:%s\n", id, err)
		return nil, errors.New(s)
	}
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

// get group id from version
func getGroupIDFromVersion(v int) (int, error) {
	var groupid int
	err := dbcon.QueryRow("select group_id from group_version where id = $1", v).Scan(&groupid)
	if err != nil {
		return 0, err
	}
	return groupid, nil
}

// update the deployed version of a group (group referred to by version!)
func updateDeployedVersionNumber(v int) error {
	gid, err := getGroupIDFromVersion(v)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid Group-Version: \"%d\": %s", v, err))
	}
	_, err = dbcon.Exec("update appgroup set deployedversion = $1 where id = $2", v, gid)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to update group: %s", err))
	}
	return nil
}

// given a version of a group checks the workers and fixes it up to match version
func applyVersion(v int) error {
	// first step: mark the version as pending
	// so if it fails for some reason, we know what to replay
	gid, err := getGroupIDFromVersion(v)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid Group-Version: \"%d\": %s", v, err))
	}
	_, err = dbcon.Exec("update appgroup set pendingversion = $1 where id = $2", v, gid)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to update group: %s", err))
	}
	apps, err := loadAppGroupVersion(v)
	if err != nil {
		return errors.New(fmt.Sprintf("error loading apps for version %d: %s", v, err))
	}
	deplid := fmt.Sprintf("%s%d", DEPLOY_PREFIX, v)
	err = MakeItSo(deplid, apps)
	if err != nil {
		return errors.New(fmt.Sprintf("error applyings apps for version %d: %s", v, err))
	}
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
	applyVersion(version)
	updateDeployedVersionNumber(version)
	r := pb.DeployResponse{}
	return &r, nil
}

func (s *DeployMonkey) UpdateApp(ctx context.Context, cr *pb.UpdateAppRequest) (*pb.GroupDefResponse, error) {
	initDB()
	if cr.Namespace == "" {
		return nil, errors.New("Namespace required")
	}
	if cr.GroupID == "" {
		return nil, errors.New("GroupID required")
	}
	if cr.App.Repository == "" {
		return nil, errors.New("App Repository required")
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
		return nil, errors.New(fmt.Sprintf("No such group: (%s,%s)", cr.Namespace, cr.GroupID))
	}
	apps, err := loadAppGroupVersion(cur.DeployedVersion)
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
	applyVersion(version)
	updateDeployedVersionNumber(version)

	return nil, errors.New("Deploy() in server - this codepath should never have been reached!")
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
