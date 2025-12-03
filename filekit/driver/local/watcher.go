package local

import (
	"github.com/fsnotify/fsnotify"
)

// fsnotifyWatcher wraps fsnotify.Watcher to implement fsWatcher interface
type fsnotifyWatcher struct {
	watcher *fsnotify.Watcher
	events  chan fsEvent
	errors  chan error
}

// newFSWatcher creates a new file system watcher using fsnotify
func newFSWatcher() (fsWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &fsnotifyWatcher{
		watcher: w,
		events:  make(chan fsEvent),
		errors:  make(chan error),
	}

	// Forward events
	go func() {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					close(fw.events)
					return
				}
				fw.events <- fsEvent{
					Name: event.Name,
					Op:   uint32(event.Op),
				}
			case err, ok := <-w.Errors:
				if !ok {
					close(fw.errors)
					return
				}
				fw.errors <- err
			}
		}
	}()

	return fw, nil
}

func (w *fsnotifyWatcher) Add(path string) error {
	return w.watcher.Add(path)
}

func (w *fsnotifyWatcher) Close() error {
	return w.watcher.Close()
}

func (w *fsnotifyWatcher) Events() <-chan fsEvent {
	return w.events
}

func (w *fsnotifyWatcher) Errors() <-chan error {
	return w.errors
}
