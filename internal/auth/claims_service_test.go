package auth

import (
	"errors"
	"testing"
)

var testSecret = []byte("test-secret-key")

func buildToken(t *testing.T, userID string) string {
	t.Helper()
	svc := NewClaimsService()
	claims := svc.CreateClaims(userID)
	tok, err := svc.BuildJWTString(claims, testSecret)
	if err != nil {
		t.Fatalf("BuildJWTString: %v", err)
	}
	return tok
}

func TestClaimsService_CreateClaims(t *testing.T) {
	svc := NewClaimsService()
	claims := svc.CreateClaims("user-42")
	if claims.UserID != "user-42" {
		t.Errorf("want UserID %q, got %q", "user-42", claims.UserID)
	}
}

func TestClaimsService_BuildJWTString(t *testing.T) {
	svc := NewClaimsService()
	claims := svc.CreateClaims("user-1")
	tok, err := svc.BuildJWTString(claims, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == "" {
		t.Error("expected non-empty token string")
	}
}

func TestClaimsService_GetUserIDFromClaims(t *testing.T) {
	svc := NewClaimsService()
	claims := &Claims{UserID: "user-99"}
	if got := svc.GetUserIDFromClaims(claims); got != "user-99" {
		t.Errorf("want %q, got %q", "user-99", got)
	}
}

func TestClaimsService_GetUserIDFromJWTString(t *testing.T) {
	tests := []struct {
		name        string
		token       func() string
		wantUserID  string
		wantErrType string // "invalid", "missing", ""
	}{
		{
			name:       "valid token",
			token:      func() string { return buildToken(t, "user-7") },
			wantUserID: "user-7",
		},
		{
			name:        "malformed token",
			token:       func() string { return "not.a.token" },
			wantErrType: "invalid",
		},
		{
			name:        "wrong secret",
			token:       func() string {
				svc := NewClaimsService()
				tok, _ := svc.BuildJWTString(svc.CreateClaims("user-1"), []byte("other-secret"))
				return tok
			},
			wantErrType: "invalid",
		},
		{
			name:        "empty userID in token",
			token:       func() string { return buildToken(t, "") },
			wantErrType: "missing",
		},
	}

	svc := NewClaimsService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := tt.token()
			gotID, err := svc.GetUserIDFromJWTString(tok, testSecret)

			switch tt.wantErrType {
			case "":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if gotID != tt.wantUserID {
					t.Errorf("want %q, got %q", tt.wantUserID, gotID)
				}
			case "invalid":
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			case "missing":
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var errMissing *ErrMissingUserID
				if !errors.As(err, &errMissing) {
					t.Errorf("expected ErrMissingUserID, got %T: %v", err, err)
				}
			}
		})
	}
}
