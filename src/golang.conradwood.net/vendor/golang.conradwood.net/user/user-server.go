package main

import (
	"flag"
	"fmt"
	"golang.conradwood.net/server"
	pb "golang.conradwood.net/user/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// static variables for flag parser
var (
	port = flag.Int("port", 5001, "the port to listen on")
)

// callback from the compound initialisation
func st(server *grpc.Server) error {
	s := new(UserAttributeServer)
	// Register the handler object
	pb.RegisterUserAttributeServiceServer(server, s)
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
* implementing the functions here:
***********************************/
type UserAttributeServer struct {
	wtf int
}

// in C we put methods into structs and call them pointers to functions
// in java/python we also put pointers to functions into structs and but call them "objects" instead
// in Go we don't put functions pointers into structs, we "associate" a function with a struct.
// (I think that's more or less the same as what C does, just different Syntax)
func (s *UserAttributeServer) GetUserDetail(ctx context.Context, pr *pb.GetUserDetailRequest) (*pb.GetUserDetailResponse, error) {
	resp, err := getUserFromDB(pr.UserID)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
func (s *UserAttributeServer) SetUserDetail(ctx context.Context, pr *pb.SetUserDetailRequest) (*pb.EmptyResponse, error) {
	return &pb.EmptyResponse{}, nil
}
func (s *UserAttributeServer) CreateUser(ctx context.Context, pr *pb.CreateUserRequest) (*pb.GetUserDetailResponse, error) {
	resp, err := getUserFromDB("foobarid")
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *UserAttributeServer) AddUserToken(ctx context.Context, pr *pb.AddTokenRequest) (*pb.EmptyResponse, error) {
	return &pb.EmptyResponse{}, nil
}

func getUserFromDB(userid string) (*pb.GetUserDetailResponse, error) {
	au := &pb.BasicUserInfo{
		FirstName: "john",
		LastName:  "doe",
		Email:     "john.doe@facebook.com",
	}
	res := &pb.GetUserDetailResponse{UserID: userid,
		UserInfo: au}
	return res, nil
}
