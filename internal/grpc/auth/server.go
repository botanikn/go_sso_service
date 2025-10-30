package auth

import (
	ssov1 "github.com/botanikn/protos/gen/go/sso"
	"google.golang.org/grpc"
	"context"
	"time"
)

type serverAPI struct {
	ssov1.UnimplementedAuthServer
}

func Register(gRPC *grpc.Server) {
	ssov1.RegisterAuthServer(gRPC, &serverAPI{})
}

func (s *serverAPI) Login(
	ctx context.Context, 
	req *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {

	// resultChan := make(chan string, 1)
	// select {
	// case <-resultChan:
	// case <-time.After(20 * time.Second):
	// }
	return &ssov1.LoginResponse{
		Token: req.Email + "_token",
	}, nil
}

func (s *serverAPI) Register(
	ctx context.Context, 
	req *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {
	panic("not implemented")
}

func (s *serverAPI) IsAdmin(
	ctx context.Context, 
	req *ssov1.IsAdminRequest,
) (*ssov1.IsAdminResponse, error) {
	panic("not implemented")
}