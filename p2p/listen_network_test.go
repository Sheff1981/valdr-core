package p2p

import "testing"

func TestListenNetwork(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{name: "ipv4 wildcard", address: "0.0.0.0:17333", want: "tcp4"},
		{name: "ipv4 loopback", address: "127.0.0.1:17333", want: "tcp4"},
		{name: "ipv6 wildcard", address: "[::]:17333", want: "tcp6"},
		{name: "ipv6 loopback", address: "[::1]:17333", want: "tcp6"},
		{name: "hostname", address: "node.example.org:17333", want: "tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenNetwork(tt.address); got != tt.want {
				t.Fatalf("listenNetwork(%q)=%q want %q", tt.address, got, tt.want)
			}
		})
	}
}
