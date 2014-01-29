package plugin

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"code.google.com/p/go.exp/fsnotify"
)

type EventType uint8

const (
	Added EventType = iota
	Removed
	Updated
)

type Walker func(string) EventHandle
type EventHandle func(EventType) error

type Loader struct {
	Plugins map[string]EventHandle
	watch   *fsnotify.Watcher
	dir     string
	mu      sync.RWMutex
}

// TODO watch a load.dir
func NewLoader(dir string) (load *Loader, err error) {
	fi, err := os.Stat(dir)
	if err != nil {
		return
	}
	if !fi.IsDir() {
		err = errors.New("loader: " + dir + " is not a directory")
		return
	}
	load = &Loader{
		Plugins: make(map[string]EventHandle),
		dir:     dir,
	}
	load.watch, err = fsnotify.NewWatcher()
	return
}

// TODO mock logging
func (load *Loader) Walk(w Walker) (err error) {
	exes, err := ioutil.ReadDir(load.dir)
	if err != nil {
		return
	}
	for _, exe := range exes {
		if !exe.IsDir() && !strings.HasSuffix(exe.Name(), ".log") {
			key := strings.ToLower(exe.Name())
			load.mu.RLock()
			_, ok := load.Plugins[key]
			load.mu.RUnlock()
			if !ok {
				log.Println("loader: ignoring duplicated " + exe.Name())
				continue
			}
			// TODO
		} else {
			log.Println("loader: ignoring " + exe.Name())
		}
	}
	return
}
