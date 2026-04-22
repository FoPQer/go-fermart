package util

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func newEchoContext() *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestGetUserIDFromToken(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(c *echo.Context)
		wantUserID string
		wantErr    bool
	}{
		{
			name: "success",
			setupCtx: func(c *echo.Context) {
				c.Set("user", &jwt.Token{Claims: jwt.MapClaims{"UserID": "user-42"}})
			},
			wantUserID: "user-42",
		},
		{
			name:     "no token in context",
			setupCtx: func(c *echo.Context) {},
			wantErr:  true,
		},
		{
			name: "claims not MapClaims",
			setupCtx: func(c *echo.Context) {
				c.Set("user", &jwt.Token{Claims: jwt.RegisteredClaims{}})
			},
			wantErr: true,
		},
		{
			name: "userID missing in claims",
			setupCtx: func(c *echo.Context) {
				c.Set("user", &jwt.Token{Claims: jwt.MapClaims{}})
			},
			wantErr: true,
		},
		{
			name: "userID wrong type in claims",
			setupCtx: func(c *echo.Context) {
				c.Set("user", &jwt.Token{Claims: jwt.MapClaims{"UserID": 123}})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newEchoContext()
			tt.setupCtx(ctx)

			gotID, err := GetUserIDFromToken(ctx)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if gotID != tt.wantUserID {
					t.Errorf("want %q, got %q", tt.wantUserID, gotID)
				}
			}
		})
	}
}
