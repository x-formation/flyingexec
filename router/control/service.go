package control

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/rjeczalik/flyingexec/util"
)

var errReq = errors.New(`router: register request ill-formed`)
var errVersion = errors.New(`control: plugin version empty`)

type res struct {
	name string
	ver  string
	err  error
	port uint16
}

// TODO reflect (extractor helper?)
func unpack(req map[string]interface{}) (id uint16, r res) {
	v, ok := req["ID"]
	if !ok {
		r.err = errReq
		return
	}
	if id, ok = v.(uint16); !ok || id == 0 {
		r.err = errReq
		return
	}
	v, ok := req["Port"]
	if !ok {
		r.err = errReq
		return
	}
	if r.port, ok = v.(uint16); !ok || r.port == 0 {
		r.err = errReq
		return
	}
	v, ok := req["Service"]
	if !ok {
		r.err = errReq
		return
	}
	if r.name, ok = v.(string); !ok || r.name == "" {
		r.err = errReq
		return
	}
	return
}

// TODO logging
type Service struct {
	pen map[uint16]chan<- res
	lis net.Listener
	mu  sync.RWMutex
}

func newService() (srvc *Service, err error) {
	srvc = &Service{
		pen: make(map[uint16]func(error)),
	}
	srvc.lis, err = util.DefaultNet.Listen("tcp", "localhost:0")
	go srvc.serve()
	return
}

// TODO handle dups
func (srvc *Service) addPen(id uint16, ch chan<- res) error {
	srvc.mu.Lock()
	srvc.pen[id] = ch
	srvc.mu.Unlock()
}

func (srvc *Service) remPen(id uitn16) {
	srvc.mu.Lock()
	delete(srvc.pen, id)
	srvc.mu.Unlock()
}

func (srvc *Service) serve() {
	srv := rpc.NewServer()
	srv.RegisterName("__ControlService", ctrl)
	log.Println("router control service listening on", srvc.lis.Addr().String(), ". . .")
	for {
		conn, err := ctrl.lis.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.ServeConn(conn)
	}
}

// TODO logging
func (srvc *Service) Register(req map[string]interface{}, _ *struct{}) error {
	id, r := unpack(req)
	if r.err != nil {
		return r.err
	}
	srvc.mu.RLock()
	ch, ok := srvc.pen[r.id]
	srvc.mu.RUnlock()
	if !ok {
		return errReq
	}
	defer func() { ch <- r }()
	conn, err := util.DefaultNet.Dial("tcp", "localhost:"+strconv.Itoa(int(r.port)))
	if err != nil {
		return err
	}
	cli := rpc.NewClient(conn)
	defer cli.Close()
	return cli.Call(r.name+".Init", srvc.lis.Addr().String(), &r.ver)
}
