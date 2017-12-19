package main

import (
	"fmt"
	"google.golang.org/grpc"
	//	"github.com/golang/protobuf/proto"
	"database/sql"
	"errors"
	"flag"
	_ "github.com/lib/pq"
	pb "golang.conradwood.net/logservice/proto"
	"golang.conradwood.net/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
	"net"
	"os"
)

// static variables for flag parser
var (
	port   = flag.Int("port", 10000, "The server port")
	dbhost = flag.String("dbhost", "postgres", "hostname of the postgres database rdms")
	dbdb   = flag.String("database", "logservice", "database to use for authentication")
	dbuser = flag.String("dbuser", "root", "username for the database to use for authentication")
	dbpw   = flag.String("dbpw", "pw", "password for the database to use for authentication")

	dbcon *sql.DB
)

// callback from the compound initialisation
func st(server *grpc.Server) error {
	s := new(LogService)
	// Register the handler object
	pb.RegisterLogServiceServer(server, s)
	return nil
}

func main() {
	var err error
	flag.Parse() // parse stuff. see "var" section above
	sd := server.NewServerDef()
	sd.Port = *port
	sd.Register = st
	dbinfo := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=require",
		*dbhost, *dbuser, *dbpw, *dbdb)
	dbcon, err = sql.Open("postgres", dbinfo)
	if err != nil {
		fmt.Printf("Failed to connect to %s on host \"%s\" as \"%s\"\n", dbdb, dbhost, dbuser)
		os.Exit(10)
	}

	err = server.ServerStartup(sd)
	if err != nil {
		fmt.Printf("failed to start server: %s\n", err)
	}
	fmt.Printf("Done\n")
	return

}

/**********************************
* implementing the functions here:
***********************************/
type LogService struct{}

/***************************************************************************************
******** BIG FAT WARNING    ----- READ ME --------
******** BIG FAT WARNING    ----- READ ME --------

* here's a funny one:
* if you print to stdout here, then it will be echoed back to you
* creating an endless loop.
* that's because we are also running in a service that logs
* stdout to us

******** BIG FAT WARNING    ----- READ ME --------
******** BIG FAT WARNING    ----- READ ME --------
***************************************************************************************/
func (s *LogService) LogCommandStdout(ctx context.Context, lr *pb.LogRequest) (*pb.LogResponse, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("Error getting peer ")
	}
	peerhost, _, err := net.SplitHostPort(peer.Addr.String())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Invalid peer: %v", peer))
	}

	user := server.GetUserID(ctx).UserID

	//fmt.Printf("%s@%s called LogCommandStdout\n", user, peerhost)
	for _, ll := range lr.Lines {
		line := ll.Line
		if len(line) > 999 {
			line = line[0:999]
		}
		_, err := dbcon.Exec("INSERT INTO logentry (loguser,peerhost,occured,status,appname,repository,namespace,groupname,deployment_id,startup_id,line) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
			user, peerhost, ll.Time, lr.AppDef.Status,
			lr.AppDef.Appname, lr.AppDef.Repository,
			lr.AppDef.Namespace, lr.AppDef.Groupname,
			lr.AppDef.DeploymentID, lr.AppDef.StartupID, line)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to log line: %s\n", err))
		}
	}
	resp := pb.LogResponse{}
	return &resp, nil
}

/***************************************************************************************
******** BIG FAT WARNING    ----- READ ME --------
******** BIG FAT WARNING    ----- READ ME --------

* here's a funny one:
* if you print to stdout here, then every time a client will tail -f our logs
* then it'll be an endless loop of following the output for this function
* basically, tail -f calls this function, so don't output to stdout

******** BIG FAT WARNING    ----- READ ME --------
******** BIG FAT WARNING    ----- READ ME --------
***************************************************************************************/
func (s *LogService) GetLogCommandStdout(ctx context.Context, lr *pb.GetLogRequest) (*pb.GetLogResponse, error) {
	var err error

	// but do take care of the minid
	minid := lr.MinimumLogID
	//fmt.Printf("Get log from minimum id: %d\n", minid)
	where := ""
	limit := int64(1000)
	if minid > 0 {
		where = fmt.Sprintf("WHERE (id > %d)", minid)
	} else if minid < 0 {
		limit = 0 - minid
		where = "WHERE (id > 0)"
	}
	// where clause for ID has been set, so we only append with AND statements to filter further

	for _, lf := range lr.LogFilter {
		if lf.Host != "" {
			return nil, errors.New("Cannot yet filter on host")
		}
		if lf.UserName != "" {
			return nil, errors.New("Cannot yet filter on userName")
		}
		if lf.AppDef == nil {
			return nil, errors.New("Cannot yet filter with empty appdef")
		}
		ad := lf.AppDef
		if ad.Status != "" {
			return nil, errors.New("Cannot yet filter on app status")
		}
		if ad.DeploymentID != "" {
			return nil, errors.New("Cannot yet filter on app deploymentid")
		}
		if ad.StartupID != "" {
			return nil, errors.New("Cannot yet filter on app startupid")
		}

		if ad.Appname != "" {
			where = fmt.Sprintf("%s AND (appname = '%s')", where, ad.Appname)
		}
		if ad.Repository != "" {
			where = fmt.Sprintf("%s AND (repository = '%s')", where, ad.Repository)
		}
		if ad.Groupname != "" {
			where = fmt.Sprintf("%s AND (groupname = '%s')", where, ad.Groupname)
		}
		if ad.Namespace != "" {
			where = fmt.Sprintf("%s AND (groupname = '%s')", where, ad.Namespace)
		}

	}
	sqlstring := fmt.Sprintf("SELECT id,loguser,peerhost,occured,status,appname,repository,namespace,groupname,deployment_id,startup_id,line from logentry %s order by id desc limit %d", where, limit)
	rows, err := dbcon.Query(sqlstring)
	defer rows.Close()
	if err != nil {
		fmt.Printf("Failed to query \"%s\": %s", sqlstring, err)
		return nil, err
	}
	response := pb.GetLogResponse{}
	i := 0
	for rows.Next() {
		i++
		ad := pb.LogAppDef{}
		le := pb.LogEntry{AppDef: &ad}
		err = rows.Scan(&le.ID, &le.UserName, &le.Host, &le.Occured,
			&le.AppDef.Status,
			&le.AppDef.Appname,
			&le.AppDef.Repository,
			&le.AppDef.Namespace,
			&le.AppDef.Groupname,
			&le.AppDef.DeploymentID,
			&le.AppDef.StartupID,
			&le.Line,
		)
		if err != nil {
			return nil, err
		}
		// since we're ordering by DESC, insert reverse order
		response.Entries = append([]*pb.LogEntry{&le}, response.Entries...)
	}
	//fmt.Printf("Returing %d log entries\n", i)
	return &response, nil
}
