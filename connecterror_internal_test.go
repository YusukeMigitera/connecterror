package connecterror

import "testing"

func TestIsConnectPackage(t *testing.T) {
	tests := []struct {
		pkgPath string
		want    bool
	}{
		{"connectrpc.com/connect", true},
		{"example.com/app/vendor/connectrpc.com/connect", true},
		{"vendor/connectrpc.com/connect", true},
		{"example.com/vendor/a/vendor/connectrpc.com/connect", true},

		{"xxxconnectrpc.com/connect", false},
		{"example.com/connectrpc.com/connect", false},
		{"connectrpc.com/connect/internal", false},
		{"connectrpc.com/connectx", false},
		{"example.com/app/vendor/xxxconnectrpc.com/connect", false},
		{"example.com/app/vendor/connectrpc.com/connect/internal", false},
		{"github.com/bufbuild/connect-go", false},
	}

	for _, tt := range tests {
		if got := isConnectPackage(tt.pkgPath); got != tt.want {
			t.Errorf("isConnectPackage(%q) = %v, want %v", tt.pkgPath, got, tt.want)
		}
	}
}
