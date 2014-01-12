package util

import (
	"testing"
)

func TestSplitHostPort(t *testing.T) {
	table := []struct {
		hostport string
		host     string
		port     uint16
	}{
		{"[::1]:59433", "::1", 59433},
		{":61200", "", 61200},
		{":0", "", 0},
		{"localhost:52302", "localhost", 52302},
	}
	tableErr := []string{
		"[::1]:67123",
		"localhost:-1",
		"localhost:e042w",
		"[::1]:",
	}
	for _, row := range table {
		host, port, err := SplitHostPort(row.hostport)
		if err != nil {
			t.Errorf("expected err to be nil, got %v instead (%q)", err, row.hostport)
		}
		if host != row.host {
			t.Errorf("expected host to be %q, got %q instead (%q)", row.host, host, row.hostport)
		}
		if port != row.port {
			t.Errorf("expected port to be %d, got %d instead (%q)", row.port, port, row.hostport)
		}
	}
	for _, hostport := range tableErr {
		if _, _, err := SplitHostPort(hostport); err == nil {
			t.Errorf("expected err to be non-nil (%q)", hostport)
		}
	}
}
