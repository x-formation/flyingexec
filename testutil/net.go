package testutil

import (
	"errors"
	"net"
	"sync"

	"github.com/rjeczalik/flyingexec/util"
)

var errClosing = errors.New("testutil: use of closed network connection")
var errRefused = errors.New("testutil: connection refused")
var errUsing = errors.New("testutil: address already in use")

type streamListener struct {
	addr net.Addr
	conn chan net.Conn
}

func newStreamListener(port uint16) *streamListener {
	return &streamListener{
		addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(port)},
		conn: make(chan net.Conn, 1),
	}
}

func (l *streamListener) Accept() (net.Conn, error) {
	conn, ok := <-l.conn
	if !ok {
		return nil, errClosing
	}
	return conn, nil
}

func (l *streamListener) Close() (err error) {
	close(l.conn)
	return
}

func (l *streamListener) Addr() net.Addr {
	return l.addr
}

type inMemNet struct {
	mu        sync.RWMutex
	listeners map[uint16]*streamListener
	counter   util.Counter
}

func (n *inMemNet) portNum(address string) (port uint16, err error) {
	if _, port, err = util.SplitHostPort(address); err != nil {
		return
	}
	if port == 0 {
		port = uint16(n.counter.Next())
		return
	}
	return
}

func (n *inMemNet) Dial(_, address string) (net.Conn, error) {
	port, err := n.portNum(address)
	if err != nil {
		return nil, err
	}
	n.mu.RLock()
	l, ok := n.listeners[port]
	n.mu.RUnlock()
	if !ok {
		return nil, errRefused
	}
	r, w := net.Pipe()
	l.conn <- r
	return w, nil
}

func (n *inMemNet) Listen(_, address string) (net.Listener, error) {
	port, err := n.portNum(address)
	if err != nil {
		return nil, err
	}
	n.mu.RLock()
	_, ok := n.listeners[port]
	n.mu.RUnlock()
	if ok {
		return nil, errUsing
	}
	l := newStreamListener(port)
	n.mu.Lock()
	n.listeners[port] = l
	n.mu.Unlock()
	return l, nil
}

var InMemNet util.Net = &inMemNet{
	listeners: make(map[uint16]*streamListener),
	counter:   1,
}
