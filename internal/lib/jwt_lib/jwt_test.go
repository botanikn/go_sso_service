package jwt_lib

import (
	"strings"
	"testing"
	"time"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const issuer = "test-issuer"

var (
	user = models.User{ID: 7, Email: "test@gmail.com"}
	app  = models.App{ID: 1, Name: "app", Secret: strings.Repeat("a", 32)}
)

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.Claims, key any) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)
	return token
}

func validClaims() Claims {
	now := time.Now()
	return Claims{
		UID:   user.ID,
		Email: user.Email,
		AppID: app.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "7",
			Audience:  jwt.ClaimStrings{"app:1"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

func TestRoundTrip(t *testing.T) {
	p := New(issuer)

	token, err := p.NewToken(user, app, time.Hour)
	require.NoError(t, err)

	userId, err := p.ParseToken(token, app)
	require.NoError(t, err)
	assert.Equal(t, user.ID, userId)
}

func TestNewToken_RejectsWeakSecret(t *testing.T) {
	_, err := New(issuer).NewToken(user, models.App{ID: 1, Secret: "short"}, time.Hour)
	assert.Error(t, err)
}

func TestParseToken_Expired(t *testing.T) {
	p := New(issuer)
	token, err := p.NewToken(user, app, time.Minute)
	require.NoError(t, err)

	p.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	_, err = p.ParseToken(token, app)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestParseToken_Rejects(t *testing.T) {
	secret := []byte(app.Secret)
	otherApp := models.App{ID: 2, Secret: strings.Repeat("b", 32)}

	noExp := validClaims()
	noExp.ExpiresAt = nil

	wrongIssuer := validClaims()
	wrongIssuer.Issuer = "evil"

	wrongAppId := validClaims()
	wrongAppId.AppID = 2

	wrongSubject := validClaims()
	wrongSubject.Subject = "8"

	otherAppToken, err := New(issuer).NewToken(user, otherApp, time.Hour)
	require.NoError(t, err)

	tests := map[string]string{
		"missing exp":   sign(t, jwt.SigningMethodHS256, noExp, secret),
		"wrong issuer":  sign(t, jwt.SigningMethodHS256, wrongIssuer, secret),
		"app_id claim":  sign(t, jwt.SigningMethodHS256, wrongAppId, secret),
		"subject claim": sign(t, jwt.SigningMethodHS256, wrongSubject, secret),
		"HS512":         sign(t, jwt.SigningMethodHS512, validClaims(), secret),
		"alg none":      sign(t, jwt.SigningMethodNone, validClaims(), jwt.UnsafeAllowNoneSignatureType),
		"wrong secret":  sign(t, jwt.SigningMethodHS256, validClaims(), []byte(strings.Repeat("c", 32))),
		"other app":     otherAppToken,
		"garbage":       "not.a.token",
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := New(issuer).ParseToken(token, app)
			assert.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}
