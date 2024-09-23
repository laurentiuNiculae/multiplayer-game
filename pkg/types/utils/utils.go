package utils

import (
	"fmt"
	"net/http"
	. "test/pkg/types"
	flatgen "test/pkg/types/flatgen/game"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
)

func NewEventHolder(kind flatgen.EventKind, event any) EventHolder {
	flatbuffEventHolder, ok := event.(FlatbuffEventHolder)
	if ok {
		return &FlatEvent{Event: flatbuffEventHolder}
	}

	eventBytes, ok := event.([]byte)
	if ok {
		return &FlatEventBytes{EventKind: kind, Event: eventBytes}
	}

	return &EmptyEvent{}
}

func NewFlatPlayerHello(builder *flatbuffers.Builder, newPlayer Player) *flatgen.PlayerHello {
	flatgen.PlayerHelloStart(builder)
	flatgen.PlayerHelloAddId(builder, int32(newPlayer.Id))
	flatgen.PlayerHelloAddKind(builder, flatgen.EventKindPlayerHello)
	flatgen.FinishPlayerHelloBuffer(builder, flatgen.PlayerHelloEnd(builder))

	return flatgen.GetRootAsPlayerHello(builder.FinishedBytes(), 0)
}

func NewFlatPlayerHelloConfirm(builder *flatbuffers.Builder, id int) *flatgen.PlayerHelloConfirm {
	flatgen.PlayerHelloConfirmStart(builder)
	flatgen.PlayerHelloConfirmAddId(builder, int32(id))
	flatgen.PlayerHelloConfirmAddKind(builder, flatgen.EventKindPlayerHelloConfirm)
	flatgen.FinishPlayerHelloConfirmBuffer(builder, flatgen.PlayerHelloConfirmEnd(builder))

	return flatgen.GetRootAsPlayerHelloConfirm(builder.FinishedBytes(), 0)
}

func NewFlatPlayerQuit(builder *flatbuffers.Builder, playerId int) *flatgen.PlayerQuit {
	flatgen.PlayerQuitStart(builder)
	flatgen.PlayerQuitAddId(builder, int32(playerId))
	flatgen.PlayerQuitAddKind(builder, flatgen.EventKindPlayerQuit)
	flatgen.FinishPlayerQuitBuffer(builder, flatgen.PlayerQuitEnd(builder))

	return flatgen.GetRootAsPlayerQuit(builder.FinishedBytes(), 0)
}

func NewFlatPlayerJoined(builder *flatbuffers.Builder, newPlayer Player) *flatgen.PlayerJoined {
	flatPlayer := NewFlatPlayer(builder, newPlayer)

	flatgen.PlayerJoinedStart(builder)
	flatgen.PlayerJoinedAddPlayer(builder, flatPlayer)
	flatgen.PlayerJoinedAddKind(builder, flatgen.EventKindPlayerJoined)
	flatgen.FinishPlayerJoinedBuffer(builder, flatgen.PlayerJoinedEnd(builder))

	return flatgen.GetRootAsPlayerJoined(builder.FinishedBytes(), 0)
}

func NewFlatPlayerMoved(builder *flatbuffers.Builder, newPlayer Player) *flatgen.PlayerMoved {
	flatPlayer := NewFlatPlayer(builder, newPlayer)

	flatgen.PlayerMovedStart(builder)
	flatgen.PlayerMovedAddPlayer(builder, flatPlayer)
	flatgen.PlayerMovedAddKind(builder, flatgen.EventKindPlayerMoved)
	flatgen.FinishPlayerMovedBuffer(builder, flatgen.PlayerMovedEnd(builder))

	return flatgen.GetRootAsPlayerMoved(builder.FinishedBytes(), 0)
}

func NewFlatPlayerMovedList(builder *flatbuffers.Builder, movingPlayers []*flatgen.PlayerMoved) *flatgen.PlayerMovedList {
	flatgen.PlayerMovedListStartPlayersVector(builder, len(movingPlayers))
	for i := range movingPlayers {
		NewFlatPlayerFromFlat(builder, movingPlayers[i].Player(nil))
	}
	movingPlayersVecOffset := builder.EndVector(len(movingPlayers))

	flatgen.PlayerMovedListStart(builder)
	flatgen.PlayerMovedListAddPlayers(builder, movingPlayersVecOffset)
	flatgen.PlayerMovedListAddKind(builder, flatgen.EventKindPlayerMovedList)
	flatgen.FinishPlayerMovedListBuffer(builder, flatgen.PlayerMovedListEnd(builder))

	return flatgen.GetRootAsPlayerMovedList(builder.FinishedBytes(), 0)
}

func NewFlatPlayerJoinedList(builder *flatbuffers.Builder, joinedPlayers []Player) *flatgen.PlayerJoinedList {
	flatgen.PlayerJoinedListStartPlayersVector(builder, len(joinedPlayers))
	for i := range joinedPlayers {
		NewFlatPlayer(builder, joinedPlayers[i])
	}
	movingPlayersVecOffset := builder.EndVector(len(joinedPlayers))

	flatgen.PlayerJoinedListStart(builder)
	flatgen.PlayerJoinedListAddPlayers(builder, movingPlayersVecOffset)
	flatgen.PlayerJoinedListAddKind(builder, flatgen.EventKindPlayerJoinedList)
	flatgen.FinishPlayerJoinedListBuffer(builder, flatgen.PlayerJoinedListEnd(builder))

	return flatgen.GetRootAsPlayerJoinedList(builder.FinishedBytes(), 0)
}

func NewFlatPlayer(builder *flatbuffers.Builder, newPlayer Player) flatbuffers.UOffsetT {
	return flatgen.CreatePlayer(builder,
		int32(newPlayer.Id),
		int32(newPlayer.X),
		int32(newPlayer.Y),
		int32(newPlayer.Speed),
		newPlayer.MovingLeft,
		newPlayer.MovingRight,
		newPlayer.MovingUp,
		newPlayer.MovingDown,
	)
}

func NewFlatPlayerFromFlat(builder *flatbuffers.Builder, newPlayer *flatgen.Player) flatbuffers.UOffsetT {
	return flatgen.CreatePlayer(builder,
		int32(newPlayer.Id()),
		int32(newPlayer.X()),
		int32(newPlayer.Y()),
		int32(newPlayer.Speed()),
		newPlayer.MovingLeft(),
		newPlayer.MovingRight(),
		newPlayer.MovingUp(),
		newPlayer.MovingDown(),
	)
}

func ParseEventBytes(data []byte) (eventKind flatgen.EventKind, eventData any, err error) {
	kindHolder := flatgen.GetRootAsKindHolder(data, 0)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("was panic, returned panic value '%v'", r)
		}
	}()

	eventKind = kindHolder.Kind()

	switch kindHolder.Kind() {
	case flatgen.EventKindPlayerHello:
		flatPlayerHello := flatgen.GetRootAsPlayerHello(data, 0)

		return eventKind, flatPlayerHello, nil
	case flatgen.EventKindPlayerHelloConfirm:
		flatPlayerHelloConfirm := flatgen.GetRootAsPlayerHelloConfirm(data, 0)

		return eventKind, flatPlayerHelloConfirm, nil
	case flatgen.EventKindPlayerQuit:
		flatPlayerQuit := flatgen.GetRootAsPlayerQuit(data, 0)

		return eventKind, flatPlayerQuit, nil
	case flatgen.EventKindPlayerJoined:
		flatPlayerJoined := flatgen.GetRootAsPlayerJoined(data, 0)

		return eventKind, flatPlayerJoined, nil
	case flatgen.EventKindPlayerJoinedList:
		flatPlayerJoinedList := flatgen.GetRootAsPlayerJoinedList(data, 0)

		return eventKind, flatPlayerJoinedList, nil
	case flatgen.EventKindPlayerMoved:
		flatPlayerMoved := flatgen.GetRootAsPlayerMoved(data, 0)

		return eventKind, flatPlayerMoved, nil
	case flatgen.EventKindPlayerMovedList:
		flatPlayerMovedList := flatgen.GetRootAsPlayerMovedList(data, 0)

		return eventKind, flatPlayerMovedList, nil
	default:
		return 0, nil, fmt.Errorf("ERROR: bogus-amogus kind '%s'", flatgen.EnumNamesEventKind[kindHolder.Kind()])
	}
}

func WaitServerIsReady(url string) {
	for {
		_, err := http.Get(url)
		if err == nil {
			return
		}

		time.Sleep(1 * time.Second)
	}
}
