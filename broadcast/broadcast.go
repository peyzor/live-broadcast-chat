package broadcast

import (
	"sync"

	"github.com/google/uuid"
)

type Listener struct {
	ID   uuid.UUID
	Chan chan string
}

type Broadcast struct {
	InputChan chan string

	lock      sync.Locker
	Listeners map[uuid.UUID]Listener
}

func NewBroadcast() *Broadcast {
	return &Broadcast{
		Listeners: make(map[uuid.UUID]Listener, 0),
		lock:      &sync.Mutex{},
	}
}

func (b *Broadcast) AddListener() Listener {
	b.lock.Lock()
	defer b.lock.Unlock()

	id, err := uuid.NewUUID()
	if err != nil {
		panic("failed to get a uuid")
	}

	l := Listener{
		ID:   id,
		Chan: make(chan string),
	}
	b.Listeners[id] = l
	return l
}

func (b *Broadcast) RemoveListener(l Listener) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.Listeners, l.ID)
}
