package main

import (
	"fmt"
	"google.golang.org/grpc"
	"time"
	//	"github.com/golang/protobuf/proto"
	"database/sql"
	"errors"
	"flag"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	pb "golang.conradwood.net/logservice/proto"
	"golang.conradwood.net/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
	"net"
	"os"
	"sort"
)

// static variables for flag parser
var (
	safeSQL = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-+=;.,%*" // exhaustive list of characters considered safe to use for sql

	port       = flag.Int("port", 10000, "The server port")
	dbhost     = flag.String("dbhost", "postgres", "hostname of the postgres database rdms")
	dbdb       = flag.String("database", "logservice", "database to use for authentication")
	dbuser     = flag.String("dbuser", "root", "username for the database to use for authentication")
	dbpw       = flag.String("dbpw", "pw", "password for the database to use for authentication")
	debug      = flag.Bool("debug", false, "turn debug output on - DANGEROUS DO NOT USE IN PRODUCTION!")
	usedb      = flag.Bool("log_to_db", true, "if false, print to std out only (not logging to database)!")
	dbcon      *sql.DB
	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logservice_requests",
			Help: "requests to log stuff received",
		},
		[]string{"appname", "groupname", "repository", "status", "user"},
	)
	lineCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logservice_lines",
			Help: "number of lines logged",
		},
		[]string{"appname", "groupname", "repository", "status", "user"},
	)
	failCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logservice_failed_requests",
			Help: "requests to log stuff received",
		},
		[]string{"appname", "groupname", "repository", "status", "user"},
	)
)

func init() {
	err := prometheus.Register(reqCounter)
	if err != nil {
		fmt.Printf("Failed to register reqCounter: %s\n", err)
	}
	err = prometheus.Register(failCounter)
	if err != nil {
		fmt.Printf("Failed to register failCounter: %s\n", err)
	}
	err = prometheus.Register(lineCounter)
	if err != nil {
		fmt.Printf("Failed to register lineCounter: %s\n", err)
	}

}

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
	if *usedb {
		dbcon, err = sql.Open("postgres", dbinfo)
		if err != nil {
			fmt.Printf("Failed to connect to %s on host \"%s\" as \"%s\"\n", dbdb, dbhost, dbuser)
			os.Exit(10)
		}
		// go SetApplicationIds()
	}
	err = server.ServerStartup(sd)
	if err != nil {
		fmt.Printf("failed to start server: %s\n", err)
	}
	fmt.Printf("Done\n")
	return

}

/**********************************
* normalize the database...
* only used during schema update
***********************************/
func SetApplicationIds() {
	for {
		fmt.Printf("Scanning 100 rows for NULL application_ids...\n")
		rows, err := dbcon.Query("select id,namespace,groupname,repository,appname from logentry where application_id is null limit 500")
		if err != nil {
			fmt.Printf("Failed to query during normalization:%s\n", err)
			return
		}
		gotSome := false
		for rows.Next() {
			var id uint64
			var n, g, r, a string
			err := rows.Scan(&id, &n, &g, &r, &a)
			if err != nil {
				fmt.Printf("Failed to scan during normalization:%s\n", err)
				return
			}
			aid, err := GetOrCreateAppID(n, g, r, a)
			if err != nil {
				fmt.Printf("Failed to get id during normalization:%s\n", err)
				return
			}
			_, err = dbcon.Exec("update logentry set application_id = $1 where id = $2", aid, id)
			if err != nil {
				fmt.Printf("Failed to update id during normalization:%s\n", err)
				return
			}
			gotSome = true
		}
		rows.Close()
		if !gotSome {
			fmt.Printf("Finished normalization. nothing more to do \n")
			return
		}
		// DEBUG -> bail out
		time.Sleep(time.Millisecond * 100)
	}
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
	l := prometheus.Labels{
		"appname":    lr.AppDef.Appname,
		"groupname":  lr.AppDef.Groupname,
		"repository": lr.AppDef.Repository,
		"status":     lr.AppDef.Status,
		"user":       user,
	}
	reqCounter.With(l).Inc()

	if *debug {
		fmt.Printf("Logging %d lines\n", len(lr.Lines))
	}
	//fmt.Printf("%s@%s called LogCommandStdout\n", user, peerhost)
	for i, ll := range lr.Lines {
		line := ll.Line
		if len(line) > 999 {
			line = line[0:999]
		}
		if *debug {
			fmt.Printf("%d of %d: \"%s\"\n", i, len(lr.Lines), line)
		}
		if *usedb {
			ad := lr.AppDef
			appid, err := GetOrCreateAppID(ad.Namespace, ad.Groupname, ad.Repository, ad.Appname)
			if err != nil {
				failCounter.With(l).Inc()
				continue
			}

			_, err = dbcon.Exec("INSERT INTO logentry (loguser,peerhost,occured,status,appname,repository,namespace,groupname,deployment_id,startup_id,line,application_id) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
				user, peerhost, ll.Time, lr.AppDef.Status,
				lr.AppDef.Appname, lr.AppDef.Repository,
				lr.AppDef.Namespace, lr.AppDef.Groupname,
				lr.AppDef.DeploymentID, lr.AppDef.StartupID, line, appid)
			/*

				_, err = dbcon.Exec("INSERT INTO logentry (loguser,peerhost,occured,status,deployment_id,startup_id,line,application_id) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
					user, peerhost, ll.Time, lr.AppDef.Status,
					lr.AppDef.DeploymentID, lr.AppDef.StartupID, line, appid)
			*/
			lineCounter.With(l).Inc()
			if err != nil {
				failCounter.With(l).Inc()
				fmt.Printf("app=%s,user=%s: Failed to log a line: %s (%s)\n", lr.AppDef.Appname, user, err, line)
				// we ignore the error and continue
				// otherwise it will be sent over, and over, and over again

				//return nil, errors.New(fmt.Sprintf("Failed to log line: %s\n", err))
			}
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
		where = fmt.Sprintf("AND (logentry.id > %d)", minid)
	} else if minid < 0 {
		limit = 0 - minid
		where = "AND (logentry.id > 0)"
	}
	// where clause for ID has been set, so we only append with AND statements to filter further
	fs, err := BuildAppFilter(lr.LogFilter)
	if err != nil {
		return nil, fmt.Errorf("Filter error:%s", err)
	}
	sqlstring := fmt.Sprintf("SELECT application.namespace,application.groupname,application.repository,application.appname,logentry.id,loguser,peerhost,occured,status,deployment_id,startup_id,line from logentry,application where logentry.application_id = application.id %s %s order by logentry.id desc limit %d", where, fs, limit)
	if *debug {
		fmt.Printf("Select: \"%s\"\n", sqlstring)
	}
	rows, err := dbcon.Query(sqlstring)
	if err != nil {
		fmt.Printf("Failed to query \"%s\": %s", sqlstring, err)
		return nil, err
	}
	defer rows.Close()
	response := &pb.GetLogResponse{}
	i := 0
	entryMap := map[int]*pb.LogEntry{}
	for rows.Next() {
		ad := pb.LogAppDef{}
		le := pb.LogEntry{AppDef: &ad}
		err = rows.Scan(&le.AppDef.Namespace,
			&le.AppDef.Groupname,
			&le.AppDef.Repository,
			&le.AppDef.Appname,
			&le.ID, &le.UserName, &le.Host, &le.Occured,
			&le.AppDef.Status,
			&le.AppDef.DeploymentID,
			&le.AppDef.StartupID,
			&le.Line,
		)
		if err != nil {
			return nil, err
		}
		entryMap[i] = &le
		i++
		// since we're ordering by DESC, insert reverse order
		//		response.Entries = append([]*pb.LogEntry{&le}, response.Entries...)
	}
	if *debug {
		fmt.Printf("Returning %d log entries\n", i)
	}
	SortResponseEntries(response, entryMap)

	return response, nil
}

func SortResponseEntries(response *pb.GetLogResponse, entryMap map[int]*pb.LogEntry) {

	response.Entries = make([]*pb.LogEntry, len(entryMap))

	// move the map contents to a fixed length array
	for x, y := range entryMap {
		response.Entries[x] = y
	}

	sort.Slice(
		response.Entries,
		func(j, k int) bool {
			return response.Entries[j].Occured < response.Entries[k].Occured
		},
	)
}

// it's not  very elegant - it returns an sql clause of numbers
// but it's better than before
func BuildAppFilter(lfs []*pb.LogFilter) (string, error) {
	ids := make(map[int]int)
	for _, lf := range lfs {
		if lf.Host != "" {
			return "", errors.New("Cannot yet filter on host")
		}
		if lf.UserName != "" {
			return "", errors.New("Cannot yet filter on userName")
		}
		if lf.AppDef == nil {
			return "", errors.New("Cannot yet filter with empty appdef")
		}
		ad := lf.AppDef
		if ad.Status != "" {
			return "", errors.New("Cannot yet filter on app status")
		}
		if ad.DeploymentID != "" {
			return "", errors.New("Cannot yet filter on app deploymentid")
		}
		if ad.StartupID != "" {
			return "", errors.New("Cannot yet filter on app startupid")
		}

		// we are concatenating sql - so we must really check for dodgy characters
		if !checkSafeSQL(ad.Appname) {
			return "", fmt.Errorf("appname contains invalid character")
		}
		if !checkSafeSQL(ad.Repository) {
			return "", fmt.Errorf("repository contains invalid character")
		}
		if !checkSafeSQL(ad.Groupname) {
			return "", fmt.Errorf("groupname contains invalid character")
		}
		if !checkSafeSQL(ad.Namespace) {
			return "", fmt.Errorf("namespace contains invalid character")
		}

		res := "appname is not null"
		gotCriteria := false
		if ad.Appname != "" {
			gotCriteria = true
			res = fmt.Sprintf("%s AND (appname = '%s')", res, ad.Appname)
		}
		if ad.Repository != "" {
			gotCriteria = true
			res = fmt.Sprintf("%s AND (repository = '%s')", res, ad.Repository)
		}
		if ad.Groupname != "" {
			gotCriteria = true
			res = fmt.Sprintf("%s AND (groupname = '%s')", res, ad.Groupname)
		}
		if ad.Namespace != "" {
			gotCriteria = true
			res = fmt.Sprintf("%s AND (namespace = '%s')", res, ad.Namespace)
		}

		if !gotCriteria {
			continue
		}
		rows, err := dbcon.Query(fmt.Sprintf("SELECT id from application where %s", res))
		if err != nil {
			return "", err
		}
		b := false
		for rows.Next() {
			b = true
			var id int
			err = rows.Scan(&id)
			if err != nil {
				rows.Close()
				return "", err
			}
			ids[id] = id
		}
		rows.Close()
		if !b {
			return "", fmt.Errorf("No exact match found for: %s", res)
		}
	}
	// array "ids" now has a list of ids from appname and groupname and friends
	if *debug {
		fmt.Printf("Non-Fuzzy match resulted in %d applications\n", len(ids))
	}

	// add fuzzymatches
	for _, lf := range lfs {
		if lf.FuzzyMatch == "" {
			continue
		}
		f := fmt.Sprintf("%%%s%%", lf.FuzzyMatch)
		if *debug {
			fmt.Printf("Fuzzymatch on \"%s\"\n", f)
		}
		rows, err := dbcon.Query("SELECT id from application where repository like $1 or namespace like $1 or groupname like $1 or appname like $1", f)
		//		rows, err := dbcon.Query("SELECT id from application where repository like $1", lf.FuzzyMatch)
		if err != nil {
			return "", fmt.Errorf("fuzzymatch error: %s", err)
		}
		b := false
		for rows.Next() {
			b = true
			var id int
			err = rows.Scan(&id)
			if err != nil {
				rows.Close()
				return "", err
			}
			ids[id] = id
		}
		rows.Close()
		if !b {
			return "", fmt.Errorf("No fuzzy match found for the filter: \"%s\"", lf.FuzzyMatch)
		}

	}
	if len(ids) == 0 {
		return "", nil
	}
	res := " AND ("
	x := ""
	for _, id := range ids {
		res = fmt.Sprintf("%s%sapplication_id = %d", res, x, id)
		x = " OR "
	}
	res = fmt.Sprintf("%s )", res)
	if *debug {
		fmt.Printf("FILTER=\"%s\"\n", res)
	}
	return res, nil
}

func checkSafeSQL(txt string) bool {
	return isOnlyChars(txt, safeSQL)
}

// true only if string "txt" is made up exclusively of characters in "valid"
func isOnlyChars(txt string, valid string) bool {
	for _, x := range txt {
		b := false
		for _, y := range valid {
			if x == y {
				b = true
			}
		}
		if !b {
			return false
		}
	}
	return true
}
func (s *LogService) GetApps(ctx context.Context, lr *pb.EmptyRequest) (*pb.GetAppsResponse, error) {
	rows, err := dbcon.Query("select namespace,groupname,appname,repository from application order by appname")
	if err != nil {
		fmt.Printf("Failed to query : %s", err)
		return nil, err
	}
	defer rows.Close()
	response := pb.GetAppsResponse{}
	for rows.Next() {
		x := pb.LogAppDef{}
		err := rows.Scan(&x.Namespace, &x.Groupname, &x.Appname, &x.Repository)
		if err != nil {
			return nil, err
		}
		response.AppDef = append(response.AppDef, &x)
	}
	return &response, nil
}
