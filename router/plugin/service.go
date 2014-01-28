package plugin

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
	service string
	ver     string
	err     error
	id      uint16
	port    uint16
}

// TODO reflect (extractor helper?)
func newRes(req map[string]interface{}) (r res, err error) {
	defer func() { err = r.err }()
	v, ok := req["ID"]
	if !ok {
		r.err = errReq
		return
	}
	if r.id, ok = v.(uint16); !ok || r.id == 0 {
		r.err = errReq
		return
	}
	v, ok = req["Port"]
	if !ok {
		r.err = errReq
		return
	}
	if r.port, ok = v.(uint16); !ok || r.port == 0 {
		r.err = errReq
		return
	}
	v, ok = req["Service"]
	if !ok {
		r.err = errReq
		return
	}
	if r.service, ok = v.(string); !ok || r.service == "" {
		r.err = errReq
		return
	}
	return
}

// TODO logging (dep inj), ability to restart
type Service struct {
	pen map[uint16]chan<- res
	lis net.Listener
	mu  sync.RWMutex
}

// TODO not ready yet
func newService() (srvc *Service, err error) {
	srvc = &Service{
		pen: make(map[uint16]chan<- res),
	}
	srvc.lis, err = util.DefaultNet.Listen("tcp", "localhost:0")
	return
}

// TODO handle dups
func (srvc *Service) addPen(id uint16, ch chan<- res) (err error) {
	srvc.mu.Lock()
	srvc.pen[id] = ch
	srvc.mu.Unlock()
	return
}

func (srvc *Service) remPen(id uint16) {
	srvc.mu.Lock()
	delete(srvc.pen, id)
	srvc.mu.Unlock()
}

// TODO able to restart, stop
func (srvc *Service) serve() {
	srv := rpc.NewServer()
	srv.RegisterName("__ControlService", srvc)
	log.Println("router control service listening on", srvc.lis.Addr().String(), ". . .")
	for {
		conn, err := srvc.lis.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.ServeConn(conn)
	}
}

// TODO rework Init to internal Init (service name, port) and external one
// (plugin dependencies, actual plugin Init)
func (srvc *Service) Register(req map[string]interface{}, _ *struct{}) (err error) {
	r, err := newRes(req)
	if err != nil {
		return
	}
	srvc.mu.RLock()
	ch, ok := srvc.pen[r.id]
	srvc.mu.RUnlock()
	if !ok {
		return errReq
	}
	defer func() {
		r.err = err
		ch <- r
	}()
	conn, err := util.DefaultNet.Dial("tcp", "localhost:"+strconv.Itoa(int(r.port)))
	if err != nil {
		return
	}
	cli := rpc.NewClient(conn)
	defer cli.Close()
	err = cli.Call(r.service+".Init", srvc.lis.Addr().String(), &r.ver)
	return
}
