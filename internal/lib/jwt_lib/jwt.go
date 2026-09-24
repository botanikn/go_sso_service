package jwt_lib

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/golang-jwt/jwt/v5"
)

const minSecretLength = 32

var (
	ErrTokenExpired = errors.New("token expired")
	ErrInvalidToken = errors.New("invalid token")
)

// Claims of an access token. uid, email and app_id are kept for backward
// compatibility with existing consumers.
type Claims struct {
	UID   int64  `json:"uid"`
	Email string `json:"email"`
	AppID int64  `json:"app_id"`
	jwt.RegisteredClaims
}

type JWTProvider struct {
	issuer string
	now    func() time.Time
}

func New(issuer string) *JWTProvider {
	return &JWTProvider{issuer: issuer, now: time.Now}
}

func (j *JWTProvider) NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	if duration <= 0 {
		return "", errors.New("duration must be positive")
	}
	if len(app.Secret) < minSecretLength {
		return "", fmt.Errorf("app secret must be at least %d bytes", minSecretLength)
	}

	jti, err := newTokenID()
	if err != nil {
		return "", err
	}

	now := j.now()
	claims := Claims{
		UID:   user.ID,
		Email: user.Email,
		AppID: app.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			Audience:  jwt.ClaimStrings{audience(app.ID)},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			ID:        jti,
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(app.Secret))
}

// ParseToken verifies the token signature and claims against the given app
// and returns the user ID it was issued for.
func (j *JWTProvider) ParseToken(tokenString string, app models.App) (int64, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(*jwt.Token) (any, error) { return []byte(app.Secret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(j.issuer),
		jwt.WithAudience(audience(app.ID)),
		jwt.WithTimeFunc(j.now),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, fmt.Errorf("%w: %w", ErrTokenExpired, err)
		}
		return 0, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if claims.AppID != app.ID {
		return 0, fmt.Errorf("%w: app_id claim mismatch", ErrInvalidToken)
	}
	if claims.UID <= 0 || claims.Subject != strconv.FormatInt(claims.UID, 10) {
		return 0, fmt.Errorf("%w: invalid subject", ErrInvalidToken)
	}

	return claims.UID, nil
}

func audience(appId int64) string {
	return "app:" + strconv.FormatInt(appId, 10)
}

func newTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
