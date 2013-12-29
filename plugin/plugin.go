package plugin

import (
	"encoding/json"
	"io"
	"net"
	"net/rpc"
	"os"
)

var stdout io.ReadWriter = os.Stdout
var stderr io.ReadWriter = os.Stderr

func Serve(rcrv interface{}) {
	encout, encerr := json.NewEncoder(stdout), json.NewEncoder(stderr)
	log := func(enc *json.Encoder, key, value string) {
		enc.Encode(map[string]string{key: value})
	}
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log(encerr, "error", err.Error())
		os.Exit(1)
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		log(encerr, "error", err.Error())
		os.Exit(1)
	}
	srv := rpc.NewServer()
	srv.Register(rcrv)
	log(encout, "port", port)
	for {
		conn, err := l.Accept()
		if err != nil {
			log(encerr, "error", err.Error())
			continue
		}
		go srv.ServeConn(conn)
	}
}
