package router

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
	"strings"

	"bitbucket.org/kardianos/osext"
	"github.com/rjeczalik/flyingexec/util"
)

type Router struct {
	Listener net.Listener
	ctrl     *control
	counter  util.Counter
}

/* TODO move to control
func (rt *Router) run(p *plugin) {
	rt.mu.Lock()
	rt.pending[p.id] = p
	rt.mu.Unlock()
	if err := p.cmd.Start(); err != nil {
		p.err = err
		rt.invalid <- p
		return
	}
	select {
	case p := <-rt.valid:
		rt.add(p)
	case <-time.After(30 * time.Second):
		rt.mu.Lock()
		p.err = errTimeout
		rt.mu.Unlock()
		rt.invalid <- p
		return
	}
	p.err = p.cmd.Wait()
}*/

func (rt *Router) routeConn(conn io.ReadWriteCloser) {
	defer conn.Close()
	var buf bytes.Buffer
	var req rpc.Request
	dec := gob.NewDecoder(io.TeeReader(conn, &buf))
	for {
		var err error
		defer func() {
			buf.Reset()
			if err != nil {
				log.Println(err)
			}
		}()
		if err = dec.Decode(&req); err != nil {
			break
		}
		if err = dec.Decode(nil); err != nil {
			break
		}
		var p *plugin
		dot := strings.LastIndex(req.ServiceMethod, ".")
		if dot > 0 {
			p, err = rt.ctrl.pluginByService(req.ServiceMethod[:dot])
		} else {
			err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		}
		if err != nil {
			break // TODO: send a response back
		}
		route, err := util.DefaultNet.Dial("tcp", p.addr)
		if err != nil {
			break // TODO: send a response back
		}
		defer route.Close()
		if _, err = route.Write(buf.Bytes()); err != nil {
			break
		}
		if _, err = io.Copy(conn, route); err != nil {
			break
		}
	}
}

func (rt *Router) ListenAndServe(addr string) (err error) {
	if rt.Listener, err = util.DefaultNet.Listen("tcp", addr); err != nil {
		return
	}
	log.Println("router listening on", rt.Listener.Addr().String(), ". . .")
	for {
		conn, err := rt.Listener.Accept()
		if err != nil {
			log.Print("error serving connection:", err)
			continue
		}
		go rt.routeConn(conn)
	}
}

func NewRouter() (rt *Router, err error) {
	rt = &Router{
		counter: 1,
	}
	execdir, err := osext.ExecutableFolder()
	if err != nil {
		return
	}
	rt.ctrl, err = newControl(execdir)
	if err != nil {
		return
	}
	return
}
