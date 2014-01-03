package plugin

import (
	"testing"

	"github.com/rjeczalik/gpf/testutil"
)

func init() {
	testutil.WatchInterrupt()
}

func TestRead(t *testing.T) {
	defer testutil.GuardPanic(t)
	table := []struct {
		input string
		err   error
		token []string
	}{
		{"8080 1", nil, []string{"8080", "1"}},
		{"33305 510 1234 135", nil, []string{"33305", "510"}},
		{"6600 0", nil, []string{"6600", "0"}},
		{"55695 43002 qwe qw", nil, []string{"55695", "43002"}},
		{"", errRead, nil},
		{"asd", errRead, nil},
		{"13123", errRead, nil},
		{"65560 123", errRead, nil},
		{"123 -1", errRead, nil},
		{"2342 qwe", errRead, nil},
	}
	for _, row := range table {
		c, err := newConnector(testutil.NewStatReader(row.input))
		if err != row.err {
			t.Errorf("expected %q, got %q instead", row.err, err)
			continue
		}
		if err == nil {
			if c.ID != row.token[0] {
				t.Errorf("expected %q, got %q instead", row.token[0], c.RouterAddr)
			}
			if c.RouterAddr != "localhost:"+row.token[1] {
				t.Errorf("expected localhost:%s, got %q instead", row.token[1], c.RouterAddr)
			}
			c.Listener.Close()
		}
	}
}
