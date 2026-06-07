package vipero

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Service testServiceConfig `json:"service"`
}

type testServiceConfig struct {
	Enabled  bool                `json:"enabled"`
	Backends []testBackendConfig `json:"backends"`
}

type testBackendConfig struct {
	BaseURL    string        `json:"baseURL"`
	Timeout    time.Duration `json:"timeout"`
	RetryCount int           `json:"retryCount"`
	RetryDelay time.Duration `json:"retryDelay"`
}

func TestDecodeSettingsLoadsObjectArrays(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")

	err := v.ReadConfig(strings.NewReader(
		"service:\n" +
			"  enabled: true\n" +
			"  backends:\n" +
			"    - baseURL: http://configured.example:8080\n" +
			"      timeout: 42s\n" +
			"      retryCount: 8\n" +
			"      retryDelay: 3s\n",
	))
	require.NoError(t, err)

	var config testConfig
	err = DecodeSettings(v, &config)
	require.NoError(t, err)
	require.Equal(t, testConfig{
		Service: testServiceConfig{
			Enabled: true,
			Backends: []testBackendConfig{
				{
					BaseURL:    "http://configured.example:8080",
					Timeout:    42 * time.Second,
					RetryCount: 8,
					RetryDelay: 3 * time.Second,
				},
			},
		},
	}, config)
}

func TestLoadConfigUsesDecodeSettings(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(configFile, []byte(
		"service:\n"+
			"  backends:\n"+
			"    - baseURL: http://configured.example:8080\n"+
			"      timeout: 42s\n"+
			"      retryCount: 8\n"+
			"      retryDelay: 3s\n",
	), 0o600)
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigType("yaml")

	var config testConfig
	err = LoadConfig(v, &config, []string{configFile})
	require.NoError(t, err)
	require.Len(t, config.Service.Backends, 1)
	require.Equal(t, "http://configured.example:8080", config.Service.Backends[0].BaseURL)
}
