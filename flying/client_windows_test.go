// +build windows

package flying

import (
	"reflect"
	"testing"
)

func TestIsService(t *testing.T) {
	cases := [...]struct {
		args []string
		name string
		cmd  []string
		ok   bool
	}{{
		[]string{"-arg1", "-arg2", "value2", "-service", "name", "-arg3", "value3"},
		"name",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		true,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-service=name", "-arg3", "value3"},
		"name",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		true,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-service", "name"},
		"name",
		[]string{"-arg1", "-arg2", "value2"},
		true,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-service=name"},
		"name",
		[]string{"-arg1", "-arg2", "value2"},
		true,
	}, {
		[]string{"-service", "name", "-arg1", "-arg2", "value2", "-arg3", "value3"},
		"name",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		true,
	}, {
		[]string{"-service=name", "-arg1", "-arg2", "value2", "-arg3", "value3"},
		"name",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		true,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-service"},
		"",
		[]string{"-arg1", "-arg2", "value2"},
		true,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-service=", "-arg3", "value3"},
		"",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		true,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		"",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "value3"},
		false,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-arg3", "-servicenot", "value3"},
		"",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "-servicenot", "value3"},
		false,
	}, {
		[]string{"-arg1", "-arg2", "value2", "-arg3", "-servicenot=value3"},
		"",
		[]string{"-arg1", "-arg2", "value2", "-arg3", "-servicenot=value3"},
		false,
	}}
	for i, cas := range cases {
		name, cmd, ok := isservice(cas.args)
		if name != cas.name || !reflect.DeepEqual(cmd, cas.cmd) || ok != cas.ok {
			t.Errorf("want name=%s, cmd=%v, ok=%v; got %s, %v, %v (i=%d)", cas.name,
				cas.cmd, cas.ok, name, cmd, ok, i)
		}
	}
}
