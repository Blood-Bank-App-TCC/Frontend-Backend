package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	adminContextKey contextKey = "admin"
	donorContextKey contextKey = "donor"

	tokenTypeAdmin = "admin"
	tokenTypeDonor = "donor"
)

type tokenClaims struct {
	Subject  string `json:"sub"`
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
	Expires  int64  `json:"exp"`
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyPassword(hash, password string) bool {
	trimmedHash := strings.TrimSpace(hash)
	if trimmedHash == "" {
		return false
	}
	if strings.HasPrefix(trimmedHash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(trimmedHash), []byte(password)) == nil
	}
	if strings.HasPrefix(trimmedHash, "sha256$") {
		sum := sha256.Sum256([]byte(password))
		return hmac.Equal([]byte(strings.TrimPrefix(trimmedHash, "sha256$")), []byte(hex.EncodeToString(sum[:])))
	}
	return hmac.Equal([]byte(trimmedHash), []byte(password))
}

func signToken(secret string, admin AdminUser, ttl time.Duration) (string, error) {
	return signClaims(secret, tokenClaims{
		Subject:  admin.ID,
		Type:     tokenTypeAdmin,
		Username: admin.Username,
		Role:     admin.Role,
		Expires:  time.Now().Add(ttl).Unix(),
	})
}

func signDonorToken(secret string, donor Donor, ttl time.Duration) (string, error) {
	return signClaims(secret, tokenClaims{
		Subject: donor.ID,
		Type:    tokenTypeDonor,
		Email:   donor.Email,
		Expires: time.Now().Add(ttl).Unix(),
	})
}

func signClaims(secret string, claims tokenClaims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	signature := hmacSHA256(secret, unsigned)
	return unsigned + "." + signature, nil
}

func parseToken(secret, token string) (AdminUser, error) {
	claims, err := parseClaims(secret, token)
	if err != nil {
		return AdminUser{}, err
	}
	if claims.Type != "" && claims.Type != tokenTypeAdmin {
		return AdminUser{}, errors.New("invalid token type")
	}
	return AdminUser{ID: claims.Subject, Username: claims.Username, Role: claims.Role}, nil
}

func parseDonorToken(secret, token string) (Donor, error) {
	claims, err := parseClaims(secret, token)
	if err != nil {
		return Donor{}, err
	}
	if claims.Type != tokenTypeDonor {
		return Donor{}, errors.New("invalid token type")
	}
	return Donor{ID: claims.Subject, Email: claims.Email}, nil
}

func parseClaims(secret, token string) (tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, errors.New("invalid token")
	}

	unsigned := parts[0] + "." + parts[1]
	expected := hmacSHA256(secret, unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return tokenClaims{}, errors.New("invalid signature")
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, err
	}

	var claims tokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return tokenClaims{}, err
	}
	if time.Now().Unix() > claims.Expires {
		return tokenClaims{}, errors.New("token expired")
	}
	return claims, nil
}

func hmacSHA256(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func adminFromContext(ctx context.Context) AdminUser {
	admin, _ := ctx.Value(adminContextKey).(AdminUser)
	return admin
}

func donorFromContext(ctx context.Context) Donor {
	donor, _ := ctx.Value(donorContextKey).(Donor)
	return donor
}

func withAdmin(ctx context.Context, admin AdminUser) context.Context {
	return context.WithValue(ctx, adminContextKey, admin)
}

func withDonor(ctx context.Context, donor Donor) context.Context {
	return context.WithValue(ctx, donorContextKey, donor)
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	if cookie, err := r.Cookie("admin_token"); err == nil {
		return cookie.Value
	}
	return ""
}
