package app

import (
	"testing"
	"time"
)

func TestHashPasswordAndVerify(t *testing.T) {
	hash, err := hashPassword("super-secret")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if hash == "super-secret" {
		t.Fatal("expected password to be hashed")
	}
	if !verifyPassword(hash, "super-secret") {
		t.Fatal("expected bcrypt password verification to succeed")
	}
	if verifyPassword(hash, "wrong-password") {
		t.Fatal("expected bcrypt password verification to fail")
	}
}

func TestDonorTokenParsingRejectsAdminToken(t *testing.T) {
	adminToken, err := signToken("test-secret", AdminUser{
		ID:       "admin-1",
		Username: "operator",
		Role:     "OPERATOR",
	}, time.Hour)
	if err != nil {
		t.Fatalf("signToken returned error: %v", err)
	}

	if _, err := parseDonorToken("test-secret", adminToken); err == nil {
		t.Fatal("expected donor token parser to reject admin token")
	}
}

func TestAdminTokenParsingRejectsDonorToken(t *testing.T) {
	donorToken, err := signDonorToken("test-secret", Donor{
		ID:    "donor-1",
		Email: "donor@example.test",
	}, time.Hour)
	if err != nil {
		t.Fatalf("signDonorToken returned error: %v", err)
	}

	if _, err := parseToken("test-secret", donorToken); err == nil {
		t.Fatal("expected admin token parser to reject donor token")
	}
}
