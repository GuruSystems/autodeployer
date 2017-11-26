package main

// see: https://grpc.io/docs/tutorials/basic/go.html

import (
	"fmt"
	//"google.golang.org/grpc"
	//	"github.com/golang/protobuf/proto"
	"flag"
	//	"net"
	pb "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
	"log"
)

// static variables for flag parser
var (
	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
	port       = flag.Int("port", 10000, "The server port")
)

func main() {
	flag.Parse()
	conn, err := client.DialWrapper("golang.conradwood.net.VpnManager")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewVpnManagerClient(conn)
	req := pb.CreateRequest{Name: "clientvpn", Access: "testaccess"}
	resp, err := cl.CreateVpn(ctx, &req)
	if err != nil {
		log.Fatalf("fail to createvpn: %v", err)
	}
	fmt.Printf("Response to createvpn: %v\n", resp)
}
