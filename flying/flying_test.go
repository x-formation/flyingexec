package flying

import (
	"os"
	"os/signal"
	"reflect"
	"testing"
	"time"
)

func discardsig() func() {
	ch := make(chan os.Signal)
	signal.Notify(ch, Signals...)
	return func() { signal.Stop(ch) }
}

// Awaiter provides an interface for (*rw.WaitingWriter).Wait().
type Awaiter interface {
	// Wait TODO
	Wait(time.Duration) error
}

type nopAwaiter struct{}

func (nopAwaiter) Wait(time.Duration) error {
	return nil
}

// Done TODO
var Done Awaiter = nopAwaiter{}

func TestMergenv(t *testing.T) {
	cases := [...]struct {
		base []string
		envs []string
		merg []string
	}{{
		[]string{"ABC=abc", "DEF=def"},
		[]string{"GHI=ghi"},
		[]string{"ABC=abc", "DEF=def", "GHI=ghi"},
	}, {
		[]string{"ABC=abc", "DEF=def", "ABC=cba"},
		[]string{"GHI=ghi"},
		[]string{"ABC=cba", "DEF=def", "GHI=ghi"},
	}, {
		[]string{"ABC=abc", "DEF=def", "GHI=old", "ABC=cba"},
		[]string{"GHI=ghi"},
		[]string{"ABC=cba", "DEF=def", "GHI=ghi"},
	}, {
		[]string{"ABC=abc", "DEF=def", "GHI=old", "ABC=cba"},
		[]string{"GHI=ghi", "DEF=fed"},
		[]string{"ABC=cba", "DEF=fed", "GHI=ghi"},
	}, {
		[]string{"ABC=abc", "DEF=def", "GHI=old", "ABC=cba", "XXX"},
		[]string{"GHI=ghi", "DEF=fed", "XYZ"},
		[]string{"ABC=cba", "DEF=fed", "GHI=ghi"},
	}, {
		os.Environ(),
		[]string{},
		os.Environ(),
	}}
	for i, cas := range cases {
		if merg := mergenv(cas.base, cas.envs...); !reflect.DeepEqual(merg, cas.merg) {
			t.Errorf("want merg=%v; got %v (i=%d)", cas.merg, merg, i)
		}
	}
}
