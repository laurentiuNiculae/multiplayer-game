package server

import (
	"iter"
	"sync"
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
	return func(yield func(int, PlayerWithSocket) bool) {
		ps.Players.Range(func(key, value any) bool {
			return yield(key.(int), value.(PlayerWithSocket))
		})
	}
}
