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
)

// static variables for flag parser
var (
	port   = flag.Int("port", 4999, "The server port")
	dbhost = flag.String("dbhost", "postgres", "hostname of the postgres database rdms")
	dbdb   = flag.String("database", "deploymonkey", "database to use for authentication")
	dbuser = flag.String("dbuser", "root", "username for the database to use for authentication")
	dbpw   = flag.String("dbpw", "pw", "password for the database to use for authentication")
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

/**********************************
* implementing the postgres functions here:
***********************************/
func initDB() error {
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
	dbcon, err := sql.Open("postgres", dbinfo)
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

// get the group with given name from database. if no such group will return nil
func getGroupFromDatabase(groupName string) (*pb.GroupDefinitionRequest, error) {
	return nil, nil

}

/**********************************
* implementing the server functions here:
***********************************/
type DeployMonkey struct {
	wtf int
}

func (s *DeployMonkey) DefineGroup(ctx context.Context, cr *pb.GroupDefinitionRequest) (*pb.EmptyResponse, error) {
	err := initDB()
	if err != nil {
		return nil, err
	}

	return nil, errors.New("Deploy() in server - this codepath should never have been reached!")
}
func (s *DeployMonkey) UpdateApp(ctx context.Context, cr *pb.UpdateAppRequest) (*pb.EmptyResponse, error) {
	initDB()
	return nil, errors.New("Deploy() in server - this codepath should never have been reached!")
}
