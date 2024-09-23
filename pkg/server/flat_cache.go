package server

import (
	"test/pkg/types"
	flatgen "test/pkg/types/flatgen/game"
	"test/pkg/types/utils"

	flatbuffers "github.com/google/flatbuffers/go"
)

type FlatCache struct {
	playerJoined map[int]*flatgen.PlayerJoined
}

func NewFlatCache() *FlatCache {
	return &FlatCache{playerJoined: map[int]*flatgen.PlayerJoined{}}
}

func (fc *FlatCache) GetMutatedPlayerJoined(id int, player types.Player) *flatgen.PlayerJoined {
	playerJoined, ok := fc.playerJoined[id]
	if !ok {
		builder := flatbuffers.NewBuilder(512)

		playerJoined = utils.NewFlatPlayerJoined(builder, player)
		fc.AddJoin(id, playerJoined)

		return playerJoined
	}

	flatPlayer := &flatgen.Player{}
	playerJoined.Player(flatPlayer)

	flatPlayer.MutateX(int32(player.X))
	flatPlayer.MutateY(int32(player.Y))
	flatPlayer.MutateSpeed(int32(player.Speed))
	flatPlayer.MutateMovingDown(player.MovingDown)
	flatPlayer.MutateMovingLeft(player.MovingLeft)
	flatPlayer.MutateMovingRight(player.MovingRight)
	flatPlayer.MutateMovingUp(player.MovingUp)

	return playerJoined
}

func (fc FlatCache) AddJoin(id int, joinEvent *flatgen.PlayerJoined) {
	fc.playerJoined[id] = joinEvent
}

func (fc FlatCache) RemoveJoin(id int) {
	delete(fc.playerJoined, id)
}
