package server_test

import (
	"fmt"
	"test/pkg/server"
	"test/pkg/types"
	"test/pkg/types/utils"
	"testing"

	flatgen "test/pkg/types/flatgen/game"

	flatbuffers "github.com/google/flatbuffers/go"
)

func TestEventCollector(t *testing.T) {
	ec := server.NewEventCollector()

	{
		builder1 := flatbuffers.NewBuilder(256)
		builder2 := flatbuffers.NewBuilder(256)

		playerJoined := utils.NewFlatPlayerJoined(builder2, types.Player{Id: 2, X: 20, Y: 69})
		playerJoinedEvent := utils.NewFlatEvent(builder1, types.PlayerJoinedKind, playerJoined.Table().Bytes)

		ec.AddEvent(2, playerJoinedEvent)
	}
	{
		builder1 := flatbuffers.NewBuilder(256)
		builder2 := flatbuffers.NewBuilder(256)

		playerJoined := utils.NewFlatPlayerJoined(builder2, types.Player{Id: 2, X: 699, Y: 420})
		playerJoinedEvent := utils.NewFlatEvent(builder1, types.PlayerJoinedKind, playerJoined.Table().Bytes)

		ec.AddEvent(2, playerJoinedEvent)
	}

	playerEvents := ec.GetPlayerEventList(2)

	{
		rawEvent := &flatgen.RawEvent{}

		fmt.Printf("playerEvents.EventsLength(): %v\n", playerEvents.EventsLength())

		playerEvents.Events(rawEvent, 0)

		event := flatgen.GetRootAsEvent(rawEvent.RawDataBytes(), 0)

		fmt.Println(event.Kind())

		playerJoined2 := flatgen.GetRootAsPlayerJoined(event.DataBytes(), 0)
		fmt.Println("Player id ", playerJoined2.Player(nil).Id())
		fmt.Println("Player X ", playerJoined2.Player(nil).X())
		fmt.Println("Player Y ", playerJoined2.Player(nil).Y())
	}
	{
		rawEvent := &flatgen.RawEvent{}

		fmt.Printf("playerEvents.EventsLength(): %v\n", playerEvents.EventsLength())

		playerEvents.Events(rawEvent, 1)

		event := flatgen.GetRootAsEvent(rawEvent.RawDataBytes(), 0)

		fmt.Println(event.Kind())

		playerJoined2 := flatgen.GetRootAsPlayerJoined(event.DataBytes(), 0)
		fmt.Println("Player id ", playerJoined2.Player(nil).Id())
		fmt.Println("Player X ", playerJoined2.Player(nil).X())
		fmt.Println("Player Y ", playerJoined2.Player(nil).Y())
	}
}

func TestEventCollectorGeneral(t *testing.T) {
	ec := server.NewEventCollector()

	playerMovedList := []*flatgen.PlayerMoved{
		utils.NewFlatPlayerMoved(flatbuffers.NewBuilder(256), types.Player{Id: 69, X: 10, Y: 200, Speed: 420}),
		utils.NewFlatPlayerMoved(flatbuffers.NewBuilder(256), types.Player{Id: 999, X: 999, Y: 999, Speed: 999}),
	}
	flatPlayerMovedList := utils.NewFlatPlayerMovedList(flatbuffers.NewBuilder(512), playerMovedList)

	ec.AddGeneralEvent(utils.NewFlatEvent(flatbuffers.NewBuilder(512), "PlayerMovedList",
		flatPlayerMovedList.Table().Bytes))
	ec.AddGeneralEvent(utils.NewFlatEvent(flatbuffers.NewBuilder(512), "PlayerMovedList",
		flatPlayerMovedList.Table().Bytes))

	ec.GetPlayerEventList(2)

	playerEvents := ec.GetPlayerEventList(2)
	fmt.Printf("list.EventsLength(): %v\n", playerEvents.EventsLength())

	rawEvent := &flatgen.RawEvent{}
	playerEvents.Events(rawEvent, 0)

	event := flatgen.GetRootAsEvent(rawEvent.RawDataBytes(), 0)

	fmt.Println(string(event.Kind()))

	playerMoved := flatgen.GetRootAsPlayerMovedList(event.DataBytes(), 0)

	player := &flatgen.Player{}
	playerMoved.Players(player, 0)

	fmt.Println("Player id ", player.Id())
	fmt.Println("Player X ", player.X())
	fmt.Println("Player Y ", player.Y())
	fmt.Println("Player Y ", player.Speed())

	playerMoved.Players(player, 1)

	fmt.Println("Player id ", player.Id())
	fmt.Println("Player X ", player.X())
	fmt.Println("Player Y ", player.Y())
	fmt.Println("Player Y ", player.Speed())
}
