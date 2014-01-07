package plugin

import (
	"strconv"
	"testing"

	"github.com/rjeczalik/flyingexec/testutil"
	"github.com/rjeczalik/flyingexec/util"
)

func init() {
	util.DefaultNet = testutil.InMemNet
	testutil.WatchInterrupt()
}

func TestRead(t *testing.T) {
	defer testutil.GuardPanic(t)
	table := []struct {
		adminPort string
		ID        string
		err       error
	}{
		{"8080", "1", nil},
		{"33305", "510", nil},
		{"6600", "0", nil},
		{"55695", "43002", nil},
		{"", "", errRead},
		{"asd", "", errRead},
		{"13123", "", errRead},
		{"65560", "123", errRead},
		{"123", "-1", errRead},
		{"2342", "qwe", errRead},
	}
	for _, row := range table {
		c, err := NewConnector(row.adminPort, row.ID)
		if err != row.err {
			t.Errorf("expected %v, got %v instead", row.err, err)
			continue
		}
		if err == nil {
			if id := strconv.Itoa(int(c.ID)); id != row.ID {
				t.Errorf("expected %q, got %q instead", row.ID, id)
			}
			if adminAddr := "localhost:" + row.adminPort; c.AdminAddr != adminAddr {
				t.Errorf("expected localhost:%d, got %v instead", adminAddr, c.AdminAddr)
			}
			c.Listener.Close()
		}
	}
}
