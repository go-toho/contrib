package corso

import (
	"net/http"
	"testing"

	"github.com/cristalhq/aconfig"
	"github.com/rs/cors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	CORS Config
}

func TestConfig_Defaults(t *testing.T) {
	var cfg TestConfig

	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		SkipDefaults: false,
		SkipFiles:    true,
		SkipEnv:      true,
		SkipFlags:    true,
	})
	err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, false, cfg.CORS.Enabled)
	assert.Equal(t, []string{"*"}, cfg.CORS.AllowedOrigins)
	assert.Equal(t, []string{"HEAD", "GET", "POST"}, cfg.CORS.AllowedMethods)
	assert.Nil(t, cfg.CORS.AllowedHeaders)
	assert.Nil(t, cfg.CORS.ExposedHeaders)
	assert.Equal(t, 0, cfg.CORS.MaxAge)
	assert.Equal(t, false, cfg.CORS.AllowCredentials)
	assert.Equal(t, false, cfg.CORS.AllowPrivateNetwork)
	assert.Equal(t, 204, cfg.CORS.OptionsSuccessStatus)
	assert.Equal(t, false, cfg.CORS.Debug)
}

func TestConfig_CORSOptions(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantOptions cors.Options
		wantEnabled bool
	}{
		{
			name: "CORS enabled with default values",
			config: Config{
				Enabled: true,
			},
			wantOptions: cors.Options{
				AllowedOrigins:       nil,
				AllowedMethods:       nil,
				AllowedHeaders:       nil,
				ExposedHeaders:       nil,
				OptionsSuccessStatus: 0,
			},
			wantEnabled: true,
		},
		{
			name: "CORS disabled",
			config: Config{
				Enabled: false,
			},
			wantOptions: cors.Options{
				OptionsSuccessStatus: http.StatusNoContent,
			},
			wantEnabled: false,
		},
		{
			name: "CORS enabled with custom values",
			config: Config{
				Enabled:              true,
				AllowedOrigins:       []string{"https://example.com", "http://localhost:3000"},
				AllowedMethods:       []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders:       []string{"Content-Type", "Authorization"},
				ExposedHeaders:       []string{"X-Custom-Header"},
				MaxAge:               3600,
				AllowCredentials:     true,
				AllowPrivateNetwork:  true,
				OptionsPassthrough:   true,
				OptionsSuccessStatus: http.StatusOK,
				Debug:                true,
			},
			wantOptions: cors.Options{
				AllowedOrigins:       []string{"https://example.com", "http://localhost:3000"},
				AllowedMethods:       []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders:       []string{"Content-Type", "Authorization"},
				ExposedHeaders:       []string{"X-Custom-Header"},
				MaxAge:               3600,
				AllowCredentials:     true,
				AllowPrivateNetwork:  true,
				OptionsPassthrough:   true,
				OptionsSuccessStatus: http.StatusOK,
				Debug:                true,
			},
			wantEnabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOptions, gotEnabled := tt.config.CORSOptions()

			if tt.wantEnabled {
				// For enabled CORS, check all fields
				assert.Equal(t, tt.wantOptions.AllowedOrigins, gotOptions.AllowedOrigins)
				assert.Equal(t, tt.wantOptions.AllowedMethods, gotOptions.AllowedMethods)
				if tt.wantOptions.AllowedHeaders == nil {
					assert.Nil(t, gotOptions.AllowedHeaders)
				} else {
					assert.Equal(t, tt.wantOptions.AllowedHeaders, gotOptions.AllowedHeaders)
				}
				if tt.wantOptions.ExposedHeaders == nil {
					assert.Nil(t, gotOptions.ExposedHeaders)
				} else {
					assert.Equal(t, tt.wantOptions.ExposedHeaders, gotOptions.ExposedHeaders)
				}
				assert.Equal(t, tt.wantOptions.MaxAge, gotOptions.MaxAge)
				assert.Equal(t, tt.wantOptions.AllowCredentials, gotOptions.AllowCredentials)
				assert.Equal(t, tt.wantOptions.AllowPrivateNetwork, gotOptions.AllowPrivateNetwork)
				assert.Equal(t, tt.wantOptions.OptionsPassthrough, gotOptions.OptionsPassthrough)
				assert.Equal(t, tt.wantOptions.OptionsSuccessStatus, gotOptions.OptionsSuccessStatus)
				assert.Equal(t, tt.wantOptions.Debug, gotOptions.Debug)
			} else {
				// For disabled CORS, just check that everything is zero/empty
				assert.Nil(t, gotOptions.AllowedOrigins)
				assert.Nil(t, gotOptions.AllowedMethods)
				assert.Nil(t, gotOptions.AllowedHeaders)
				assert.Nil(t, gotOptions.ExposedHeaders)
				assert.Equal(t, 0, gotOptions.MaxAge)
				assert.False(t, gotOptions.AllowCredentials)
				assert.False(t, gotOptions.AllowPrivateNetwork)
				assert.False(t, gotOptions.OptionsPassthrough)
				assert.Equal(t, 0, gotOptions.OptionsSuccessStatus)
				assert.False(t, gotOptions.Debug)
			}
			assert.Equal(t, tt.wantEnabled, gotEnabled)
		})
	}
}
