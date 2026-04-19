package skills

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

type WatchCallback func(path string, op string)

type Watcher struct {
	w  *fsnotify.Watcher
	cb WatchCallback
}

func NewWatcher(cb WatchCallback) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{w: w, cb: cb}, nil
}

func (wt *Watcher) Add(path string) error { return wt.w.Add(path) }

func (wt *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-wt.w.Events:
				if !ok {
					return
				}
				wt.cb(event.Name, event.Op.String())
			case err, ok := <-wt.w.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()
}

func (wt *Watcher) Stop() { _ = wt.w.Close() }
