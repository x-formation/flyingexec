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
	addr string
	serv string
	ver  string
	err  error
	id   uint32
}

func newRes(req map[string]string) (r res, err error) {
	for _, k := range []string{"ID", "Port", "Service"} {
		v, ok := req[k]
		if !ok || v == "" {
			err = errReq
			return
		}
	}
	if _, err = strconv.ParseUint(req["Port"], 10, 16); err != nil {
		return
	}
	id, err := strconv.ParseUint(req["ID"], 10, 32)
	if err != nil {
		return
	}
	r.id = uint32(id)
	r.addr = "localhost:" + req["Port"]
	r.serv = req["Service"]
	return
}

// TODO logging (dep inj), ability to restart
type Service struct {
	pen map[uint32]chan<- res
	lis net.Listener
	mu  sync.RWMutex
}

// TODO not ready yet
func newService() (srvc *Service, err error) {
	srvc = &Service{
		pen: make(map[uint32]chan<- res),
	}
	srvc.lis, err = util.DefaultNet.Listen("tcp", "localhost:0")
	return
}

// TODO handle dups
func (srvc *Service) addPen(id uint32, ch chan<- res) (err error) {
	srvc.mu.Lock()
	srvc.pen[id] = ch
	srvc.mu.Unlock()
	return
}

func (srvc *Service) remPen(id uint32) {
	srvc.mu.Lock()
	delete(srvc.pen, id)
	srvc.mu.Unlock()
}

// TODO ablility to restart, stop
// TODO abstract for-Accept loop
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
func (srvc *Service) Register(req map[string]string, _ *struct{}) (err error) {
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
	conn, err := util.DefaultNet.Dial("tcp", r.addr)
	if err != nil {
		return
	}
	cli := rpc.NewClient(conn)
	defer cli.Close()
	err = cli.Call(r.serv+".Init", srvc.lis.Addr().String(), &r.ver)
	return
}
