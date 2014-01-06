package plugin

import (
	"strconv"
	"testing"

	"github.com/rjeczalik/flyingexec/testutil"
)

func init() {
	testutil.WatchInterrupt()
}

func TestRead(t *testing.T) {
	defer testutil.GuardPanic(t)
	table := []struct {
		input string
		err   error
		id    uint16
		port  int
	}{
		{"8080 1", nil, 8080, 1},
		{"33305 510 1234 135", nil, 33305, 510},
		{"6600 0", nil, 6600, 0},
		{"55695 43002 qwe qw", nil, 55695, 43002},
		{input: "", err: errRead},
		{input: "asd", err: errRead},
		{input: "13123", err: errRead},
		{input: "65560 123", err: errRead},
		{input: "123 -1", err: errRead},
		{input: "2342 qwe", err: errRead},
	}
	for _, row := range table {
		c, err := newConnector(testutil.NewStatReader(row.input))
		if err != row.err {
			t.Errorf("expected %v, got %v instead", row.err, err)
			continue
		}
		if err == nil {
			if c.ID != row.id {
				t.Errorf("expected %v, got %v instead", row.id, c.RouterAddr)
			}
			if c.RouterAddr != "localhost:"+strconv.Itoa(row.port) {
				t.Errorf("expected localhost:%d, got %v instead", row.port, c.RouterAddr)
			}
			c.Listener.Close()
		}
	}
}
