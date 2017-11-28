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
)

// static variables for flag parser
var (
	port   = flag.Int("port", 4999, "The server port")
	dbhost = flag.String("dbhost", "postgres", "hostname of the postgres database rdms")
	dbdb   = flag.String("database", "deploymonkey", "database to use for authentication")
	dbuser = flag.String("dbuser", "root", "username for the database to use for authentication")
	dbpw   = flag.String("dbpw", "pw", "password for the database to use for authentication")
	file   = flag.String("filename", "", "filename with a group definition (for testing)")
	dbcon  *sql.DB
	dbinfo string
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
	sd.Register = st
	err := server.ServerStartup(sd)
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
		_, err := dm.DefineGroup(nil, gdr)
		if err != nil {
			fmt.Printf("Failed to manage group %s: %s\n", gdr.GroupID, err)
			return
		}
	}
	fmt.Printf("File parsed.\n")

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
	id       int
	groupDef *pb.GroupDefinitionRequest
}

// get the group with given name from database. if no such group will return nil
func getGroupFromDatabase(nameSpace string, groupName string) (*DBGroup, error) {
	var deplVers int
	res := pb.GroupDefinitionRequest{}
	d := DBGroup{}
	res.Namespace = nameSpace
	rows, err := dbcon.Query("SELECT id,groupname,deployedversion from appgroup where groupname=$1", groupName)
	if err != nil {
		fmt.Printf("Failed to get groupname %s\n", groupName)
		return nil, err
	}
	for rows.Next() {
		err := rows.Scan(&d.id, &res.GroupID, &deplVers)
		if err != nil {
			fmt.Printf("Failed to get row for groupname %s\n", groupName)
			return nil, err
		}
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
		return "", errors.New(fmt.Sprintf("Failed to insert application: %s", err))
	}
	versionId := id
	fmt.Printf("New Version: %d\n", versionId)
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
	return "", nil
}

func saveApp(app *pb.ApplicationDefinition) (string, error) {
	var id int
	err := dbcon.QueryRow("INSERT into appdef (sourceurl,downloaduser,downloadpw,executable,repo,buildid,instances) values ($1,$2,$3,$4,$5,$6,$7) RETURNING id",
		app.DownloadURL, app.DownloadUser, app.DownloadPassword,
		app.Binary, app.Repository, app.BuildID, app.Instances).Scan(&id)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to insert application: %s", err))
	}
	return fmt.Sprintf("%d", id), nil
}

/**********************************
* implementing the server functions here:
***********************************/
type DeployMonkey struct {
	wtf int
}

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
	diff, err := Compare(cur.groupDef, cr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to compare: %s", err))
	}
	if len(diff.AppDiffs) == 0 {
		r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_NOCHANGE}
		return &r, nil
	}

	for _, dg := range diff.AppDiffs {
		fmt.Printf("Update: %s\n", dg.Describe())
	}
	_, err = createGroupVersion(cr.Namespace, cr.GroupID, cr.Applications)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create new version: %s", err))
	}
	// we do have diffs, so create a new version and put the new definition in to the database
	r := pb.GroupDefResponse{Result: pb.GroupResponseStatus_CHANGEACCEPTED}
	return &r, nil
}
func (s *DeployMonkey) UpdateApp(ctx context.Context, cr *pb.UpdateAppRequest) (*pb.EmptyResponse, error) {
	initDB()
	return nil, errors.New("Deploy() in server - this codepath should never have been reached!")
}
