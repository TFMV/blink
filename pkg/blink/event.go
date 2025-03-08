package blink

import (
	"github.com/fsnotify/fsnotify"
)

type Event fsnotify.Event

func (e Event) String() string {
	return fsnotify.Event(e).String()
}
