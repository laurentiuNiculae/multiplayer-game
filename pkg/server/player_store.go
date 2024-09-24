package server

import (
	"iter"
	"sync"

	. "github.com/laurentiuNiculae/multiplayer-game/pkg/types"
)

type PlayerStore interface {
	Get(id int) (PlayerWithSocket, bool)
	Delete(id int)
	Set(id int, conn PlayerWithSocket)
	All() iter.Seq2[int, PlayerWithSocket]
}

type BasePlayerStore struct {
	Players sync.Map
}

func NewPlayerStore() *BasePlayerStore {
	return &BasePlayerStore{
		Players: sync.Map{},
	}
}

func (ps *BasePlayerStore) Get(id int) (player PlayerWithSocket, ok bool) {
	val, ok := ps.Players.Load(id)

	if !ok {
		return player, false
	}

	player = val.(PlayerWithSocket)

	return player, true
}

func (ps *BasePlayerStore) Set(id int, player PlayerWithSocket) {
	ps.Players.Store(id, player)
}

func (ps *BasePlayerStore) Delete(id int) {
	ps.Players.Delete(id)
}

func (ps *BasePlayerStore) All() iter.Seq2[int, PlayerWithSocket] {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		return
	// 	}
	// }()

	return func(yield func(int, PlayerWithSocket) bool) {
		ps.Players.Range(func(key, value any) (ok bool) {
			key, ok1 := key.(int)

			value, ok2 := value.(PlayerWithSocket)

			if ok1 && ok2 {
				return yield(key.(int), value.(PlayerWithSocket))
			}

			return true
		})
	}
}
