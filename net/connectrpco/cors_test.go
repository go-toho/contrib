package connectrpco

import (
	"slices"
	"testing"

	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"
	"github.com/stretchr/testify/assert"
)

func TestNewConnectCorsOptions(t *testing.T) {
	tests := []struct {
		name   string
		input  cors.Options
		expect cors.Options
	}{
		{
			name:  "empty options gets defaults",
			input: cors.Options{},
			expect: cors.Options{
				AllowedMethods: connectcors.AllowedMethods(),
				AllowedHeaders: connectcors.AllowedHeaders(),
				ExposedHeaders: connectcors.ExposedHeaders(),
				MaxAge:         7200,
			},
		},
		{
			name: "custom options merges with defaults",
			input: cors.Options{
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"X-Custom"},
				ExposedHeaders: []string{"X-Other"},
				MaxAge:         3600,
			},
			expect: cors.Options{
				AllowedMethods: slices.Compact(append([]string{"GET"}, connectcors.AllowedMethods()...)),
				AllowedHeaders: slices.Compact(append([]string{"X-Custom"}, connectcors.AllowedHeaders()...)),
				ExposedHeaders: slices.Compact(append([]string{"X-Other"}, connectcors.ExposedHeaders()...)),
				MaxAge:         3600,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewConnectCorsOptions(tt.input)
			assert.Equal(t, tt.expect.AllowedMethods, result.AllowedMethods)
			assert.Equal(t, tt.expect.AllowedHeaders, result.AllowedHeaders)
			assert.Equal(t, tt.expect.ExposedHeaders, result.ExposedHeaders)
			assert.Equal(t, tt.expect.MaxAge, result.MaxAge)
		})
	}
}
