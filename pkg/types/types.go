package types

import (
	flatgen "test/pkg/types/flatgen/game"

	"github.com/coder/websocket"
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
