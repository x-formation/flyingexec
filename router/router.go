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

	"github.com/rjeczalik/flyingexec/loader"
	"github.com/rjeczalik/flyingexec/router/plugin"
	"github.com/rjeczalik/flyingexec/util"

	"bitbucket.org/kardianos/osext"
)

type Router struct {
	Listener net.Listener
	Control  *plugin.Control
	Loader   *loader.Loader
}

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
		dot := strings.LastIndex(req.ServiceMethod, ".")
		if dot == 0 {
			err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
			break // TODO send a response back
		}
		pluginConn, err := rt.Control.Dial(req.ServiceMethod[:dot])
		if err != nil {
			break // TODO send a response back
		}
		defer pluginConn.Close()
		// TODO handle short write
		if _, err = pluginConn.Write(buf.Bytes()); err != nil {
			break
		}
		if _, err = io.Copy(conn, pluginConn); err != nil {
			break
		}
	}
}

// TODO abstract for-Accept loop
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
	execdir, err := osext.ExecutableFolder()
	if err != nil {
		return
	}
	rt = new(Router)
	if rt.Control, err = plugin.NewControl(); err != nil {
		return
	}
	rt.Loader, err = loader.NewLoader(execdir)
	return
}
