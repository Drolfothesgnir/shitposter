package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractHostPort(t *testing.T) {
	type tc struct {
		name      string
		addr      string
		wantHost  string
		wantPort  string
		wantError bool
	}

	tests := []tc{
		{
			name:     "with_scheme_host_and_port",
			addr:     "http://localhost:8080",
			wantHost: "localhost",
			wantPort: "8080",
		},
		{
			name:     "with_scheme_only_host",
			addr:     "http://localhost",
			wantHost: "localhost",
			wantPort: "",
		},
		{
			name:     "ipv4_with_scheme",
			addr:     "http://0.0.0.0:8080",
			wantHost: "0.0.0.0",
			wantPort: "8080",
		},
		{
			name:     "domain_with_scheme",
			addr:     "http://example.com:443",
			wantHost: "example.com",
			wantPort: "443",
		},
		{
			name:     "ipv6_with_scheme_host_and_port",
			addr:     "http://[::1]:9090",
			wantHost: "::1",
			wantPort: "9090",
		},
		{
			name:     "ipv6_with_scheme_only_host",
			addr:     "http://[::1]",
			wantHost: "::1",
			wantPort: "",
		},
		{
			name:     "no_scheme_host_and_port",
			addr:     "localhost:8080",
			wantHost: "localhost",
			wantPort: "8080",
		},
		{
			name:     "no_scheme_ipv6",
			addr:     "[::1]:9090",
			wantHost: "::1",
			wantPort: "9090",
		},
		{
			name:      "invalid_url_missing_host",
			addr:      "http://:8080",
			wantError: true,
		},
		{
			name:      "garbage_string",
			addr:      "not a url",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{HTTPServerAddress: tt.addr}
			host, port, err := cfg.ExtractHostPort()

			if tt.wantError {
				require.Error(t, err, "expected error for addr=%q", tt.addr)
				return
			}

			require.NoError(t, err, "unexpected error for addr=%q", tt.addr)
			require.Equal(t, tt.wantHost, host, "wrong host for addr=%q", tt.addr)
			require.Equal(t, tt.wantPort, port, "wrong port for addr=%q", tt.addr)
		})
	}
}
