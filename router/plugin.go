package router

import (
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"sync"

	"github.com/rjeczalik/flyingexec/util"
)

type plugin struct {
	id      uint16
	service string
	version string
	addr    string
	cmd     *exec.Cmd
	log     *os.File
}

func (p *plugin) String() string {
	cmd, log := "<invalid>", "<invalid>"
	if p.cmd != nil {
		cmd = p.cmd.Path
	}
	if p.log != nil {
		log = p.log.Name()
	}
	return fmt.Sprintf("service %q (id=%d, path=%s, log=%s, version=%s), "+
		"listening on %s", p.service, p.id, cmd, log, p.version, p.addr)
}

func (p *plugin) init(routerAddr string) (err error) {
	conn, err := util.DefaultNet.Dial("tcp", p.addr)
	if err != nil {
		return
	}
	cli := rpc.NewClient(conn)
	defer cli.Close()
	err = cli.Call(p.service+".Init", routerAddr, &p.version)
	return
}

type pluginContainer struct {
	mu      sync.RWMutex
	reg     map[string]*plugin
	pending map[uint16]*plugin
}

func newPluginContainer() *pluginContainer {
	return &pluginContainer{
		reg:     make(map[string]*plugin),
		pending: make(map[uint16]*plugin),
	}
}

func (pcnt *pluginContainer) pluginByService(name string) (p *plugin, err error) {
	pcnt.mu.RLock()
	p, ok := pcnt.reg[name]
	pcnt.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("rps: can't find service %q", name)
	}
	return
}

// TODO: add to watchlist, if a pending plugin did not get registered before
// a timeout hits, we should fail and remove it from pending
func (pcnt *pluginContainer) addPending(p *plugin) error {
	pcnt.mu.RLock()
	_, ok := pcnt.pending[p.id]
	pcnt.mu.RUnlock()
	if ok {
		return fmt.Errorf("plugin container: plugin with ID=%d is already pending", p.id)
	}
	pcnt.mu.Lock()
	pcnt.pending[p.id] = p
	pcnt.mu.RLock()
	return nil
}

func (pcnt *pluginContainer) popPending(id uint16) (*plugin, error) {
	pcnt.mu.RLock()
	p, ok := pcnt.pending[id]
	pcnt.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("plugin container: no plugin with ID=%d is pending", id)
	}
	pcnt.mu.Lock()
	delete(pcnt.pending, id)
	pcnt.mu.Unlock()
	return p, nil
}

func (pcnt *pluginContainer) addReg(p *plugin) error {
	pcnt.mu.RLock()
	_, ok := pcnt.reg[p.service]
	pcnt.mu.RUnlock()
	if ok {
		return fmt.Errorf("error adding plugin %s: service is already registered", p)
	}
	pcnt.mu.Lock()
	pcnt.reg[p.service] = p
	pcnt.mu.Unlock()
	return nil
}

// TODO
func (pcnt *pluginContainer) delReg(p *plugin) error {
	return nil
}
