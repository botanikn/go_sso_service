package auth

import (
	ssov1 "github.com/botanikn/go_sso_service/gen/go/sso"
)

type serverServer struct {
	ssov1.UnimplementedAuthServiceServer
}

func Register(gRPC *grpc.Server) {
	ssov1.RegisterAuthServer(gRPC, &serverServer{})
}

func (s *serverAPI) Login(
	ctx context.Context, 
	req *ssov1.LoginRequest
) (*ssov1.LoginResponse, error) {
	pamnic("not implemented")
}

func (s *serverAPI) Register(
	ctx context.Context, 
	req *ssov1.RegisterRequest
) (*ssov1.RegisterResponse, error) {
	pamnic("not implemented")
}

func (s *serverAPI) IsAdmin(
	ctx context.Context, 
	req *ssov1.IsAdminRequest
) (*ssov1.IsAdminResponse, error) {
	pamnic("not implemented")
}