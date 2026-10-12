package connectrpco

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/authn"
	"connectrpc.com/connect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthnMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "authn error",
			err:        authn.Errorf("bad token"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"code":"unauthenticated","message":"bad token"}`,
		},
		{
			name:       "connect v2 error",
			err:        connect.NewError(connect.CodeUnauthenticated, "bad token"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"code":"unauthenticated","message":"bad token"}`,
		},
		{
			name:       "wrapped connect v2 error",
			err:        fmt.Errorf("auth: %w", connect.NewError(connect.CodePermissionDenied, "forbidden")),
			wantStatus: http.StatusForbidden,
			wantBody:   `{"code":"permission_denied","message":"forbidden"}`,
		},
		{
			name:       "plain error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"code":"unknown","message":"boom"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := NewAuthnMiddleware(func(context.Context, *http.Request) (any, error) {
				return nil, tt.err
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/test.v1.EchoService/Echo", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			mw(http.NotFoundHandler()).ServeHTTP(rec, req)

			body, err := io.ReadAll(rec.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.JSONEq(t, tt.wantBody, string(body))
		})
	}
}

func TestNewAuthnMiddleware_Info(t *testing.T) {
	mw := NewAuthnMiddleware(func(context.Context, *http.Request) (any, error) {
		return "user-1", nil
	})

	var info any
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		info = authn.GetInfo(r.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

	assert.Equal(t, "user-1", info)
}
