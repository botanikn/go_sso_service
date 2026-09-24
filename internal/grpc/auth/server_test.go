package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/botanikn/go_sso_service/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestToStatus(t *testing.T) {
	tests := []struct {
		err     error
		code    codes.Code
		message string
	}{
		{services.InvalidArgument("invalid email"), codes.InvalidArgument, "invalid email"},
		{services.ErrInvalidCredentials, codes.Unauthenticated, "invalid email or password"},
		{services.ErrTokenExpired, codes.Unauthenticated, "token expired"},
		{services.ErrInvalidToken, codes.Unauthenticated, "invalid token"},
		{services.ErrUserBanned, codes.PermissionDenied, "user is banned"},
		{services.ErrForbidden, codes.PermissionDenied, "insufficient permissions"},
		{services.ErrUserAlreadyExists, codes.AlreadyExists, "user already exists"},
		{services.ErrAppAlreadyExists, codes.AlreadyExists, "app already exists"},
		{services.ErrAppDoesNotExist, codes.NotFound, "app not found"},
		{services.ErrPermissionDoesNotExist, codes.NotFound, "permission not found"},
		{errors.New("pq: password authentication failed"), codes.Internal, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			st := status.Convert(toStatus(fmt.Errorf("op: %w", tt.err)))
			assert.Equal(t, tt.code, st.Code())
			assert.Equal(t, tt.message, st.Message())
		})
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header []string
		want   string
		ok     bool
	}{
		{"bearer", []string{"Bearer abc"}, "abc", true},
		{"lowercase bearer", []string{"bearer abc"}, "abc", true},
		{"raw token", []string{"abc"}, "abc", true},
		{"empty bearer", []string{"Bearer "}, "", false},
		{"missing", nil, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.MD{}
			if tt.header != nil {
				md.Set("authorization", tt.header...)
			}
			token, err := bearerToken(metadata.NewIncomingContext(context.Background(), md))
			if !tt.ok {
				assert.Equal(t, codes.Unauthenticated, status.Code(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, token)
		})
	}
}
