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
			name:       "connect error",
			err:        connect.NewError(connect.CodeUnauthenticated, "bad token"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"code":"unauthenticated","message":"bad token"}`,
		},
		{
			name:       "wrapped connect error",
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
		info = GetAuthnInfo(r.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

	assert.Equal(t, "user-1", info)
}

func TestNewAuthnMiddleware_GRPC(t *testing.T) {
	mw := NewAuthnMiddleware(func(context.Context, *http.Request) (any, error) {
		return nil, connect.NewError(connect.CodeUnauthenticated, "bad token")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test.v1.EchoService/Echo", nil)
	req.Header.Set("Content-Type", "application/grpc")
	mw(http.NotFoundHandler()).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "16", rec.Header().Get("Grpc-Status"))
	assert.Equal(t, "bad token", rec.Header().Get("Grpc-Message"))
}
