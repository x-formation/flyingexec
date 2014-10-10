package flying

import (
	"os"
	"os/signal"
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
