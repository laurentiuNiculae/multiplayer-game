package types

import (
	flatgen "github.com/laurentiuNiculae/multiplayer-game/pkg/types/flatgen/game"

	"github.com/coder/websocket"
	flatbuffers "github.com/google/flatbuffers/go"
)

type Player struct {
	Id          int
	X, Y        float64
	Speed       float64
	MovingLeft  bool
	MovingRight bool
	MovingUp    bool
	MovingDown  bool
}

type PlayerWithSocket struct {
	Player
	Conn *websocket.Conn
}

type Event struct {
	Kind     flatgen.EventKind
	PlayerId int
	Conn     *websocket.Conn
	Data     any
}

type PlayerHello struct {
	Kind flatgen.EventKind
	Id   int
}

// EventHolder is used for when we send events to users, we care only for the bytes and kind
type EventHolder interface {
	Kind() flatgen.EventKind
	Bytes() []byte
}

// Generic flatbuf table

type FlatbuffEventHolder interface {
	Kind() flatgen.EventKind
	Table() flatbuffers.Table
}

type FlatEvent struct {
	Event FlatbuffEventHolder
}

func (fe *FlatEvent) Kind() flatgen.EventKind {
	return fe.Event.Kind()
}

func (fe *FlatEvent) Bytes() []byte {
	return fe.Event.Table().Bytes
}

type FlatEventBytes struct {
	EventKind flatgen.EventKind
	Event     []byte
}

func (fe *FlatEventBytes) Kind() flatgen.EventKind {
	return fe.EventKind
}

func (fe *FlatEventBytes) Bytes() []byte {
	return fe.Event
}

type EmptyEvent struct {
}

func (fe *EmptyEvent) Kind() flatgen.EventKind {
	return flatgen.EventKindNilEvent
}

func (fe *EmptyEvent) Bytes() []byte {
	return nil
}
