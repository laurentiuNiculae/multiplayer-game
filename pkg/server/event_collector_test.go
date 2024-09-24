package server

import (
	"fmt"
	"testing"

	"github.com/laurentiuNiculae/multiplayer-game/pkg/server"
	"github.com/laurentiuNiculae/multiplayer-game/pkg/types"
	"github.com/laurentiuNiculae/multiplayer-game/pkg/types/utils"

	flatgen "github.com/laurentiuNiculae/multiplayer-game/pkg/types/flatgen/game"

	flatbuffers "github.com/google/flatbuffers/go"
)

func TestEventCollector(t *testing.T) {
	ec := server.NewEventCollector()

	{
		builder2 := flatbuffers.NewBuilder(256)

		playerJoined := utils.NewFlatPlayerJoined(builder2, types.Player{Id: 2, X: 20, Y: 69})
		playerJoinedEvent := utils.NewEventHolder(flatgen.EventKindPlayerJoined, playerJoined)

		ec.AddEvent(2, playerJoinedEvent)
	}
	{
		builder2 := flatbuffers.NewBuilder(256)

		playerJoined := utils.NewFlatPlayerJoined(builder2, types.Player{Id: 2, X: 699, Y: 420})
		playerJoinedEvent := utils.NewEventHolder(flatgen.EventKindPlayerJoined, playerJoined)

		ec.AddEvent(2, playerJoinedEvent)
	}

	playerEvents, _ := ec.GetPlayerEventList(2)

	{
		rawEvent := &flatgen.RawEvent{}

		fmt.Printf("playerEvents.EventsLength(): %v\n", playerEvents.EventsLength())

		playerEvents.Events(rawEvent, 0)

		kindHolder := flatgen.GetRootAsKindHolder(rawEvent.RawDataBytes(), 0)

		fmt.Println(flatgen.EnumNamesEventKind[kindHolder.Kind()])

		playerJoined2 := flatgen.GetRootAsPlayerJoined(rawEvent.RawDataBytes(), 0)
		fmt.Println("Player id ", playerJoined2.Player(nil).Id())
		fmt.Println("Player X ", playerJoined2.Player(nil).X())
		fmt.Println("Player Y ", playerJoined2.Player(nil).Y())
	}
	{
		rawEvent := &flatgen.RawEvent{}

		fmt.Printf("playerEvents.EventsLength(): %v\n", playerEvents.EventsLength())

		playerEvents.Events(rawEvent, 1)

		kindHolder := flatgen.GetRootAsKindHolder(rawEvent.RawDataBytes(), 0)

		fmt.Println(flatgen.EnumNamesEventKind[kindHolder.Kind()])

		playerJoined2 := flatgen.GetRootAsPlayerJoined(rawEvent.RawDataBytes(), 0)
		fmt.Println("Player id ", playerJoined2.Player(nil).Id())
		fmt.Println("Player X ", playerJoined2.Player(nil).X())
		fmt.Println("Player Y ", playerJoined2.Player(nil).Y())
	}
}

func TestEventCollectorGeneral(t *testing.T) {
	ec := server.NewEventCollector()

	playerMovedList := []*flatgen.PlayerMoved{
		utils.NewFlatPlayerMoved(flatbuffers.NewBuilder(256), types.Player{Id: 69, X: 10, Y: 200, Speed: 420}),
		utils.NewFlatPlayerMoved(flatbuffers.NewBuilder(256), types.Player{Id: 989, X: 999, Y: 992, Speed: 909}),
	}
	flatPlayerMovedList := utils.NewFlatPlayerMovedList(flatbuffers.NewBuilder(512), playerMovedList)

	ec.AddGeneralEvent(utils.NewEventHolder(flatgen.EventKindPlayerMoved, flatPlayerMovedList))
	ec.AddGeneralEvent(utils.NewEventHolder(flatgen.EventKindPlayerMoved, flatPlayerMovedList))

	playerEvents, _ := ec.GetPlayerEventList(2)
	fmt.Printf("list.EventsLength(): %v\n", playerEvents.EventsLength())

	rawEvent := &flatgen.RawEvent{}
	playerEvents.Events(rawEvent, 0)

	kindHolder := flatgen.GetRootAsKindHolder(rawEvent.RawDataBytes(), 0)

	fmt.Println(flatgen.EnumNamesEventKind[kindHolder.Kind()])

	playerMoved := flatgen.GetRootAsPlayerMovedList(rawEvent.RawDataBytes(), 0)

	player := &flatgen.Player{}
	playerMoved.Players(player, 0)

	fmt.Println("Player id ", player.Id())
	fmt.Println("Player X ", player.X())
	fmt.Println("Player Y ", player.Y())
	fmt.Println("Player Speed ", player.Speed())

	playerMoved.Players(player, 1)

	fmt.Println("Player id ", player.Id())
	fmt.Println("Player X ", player.X())
	fmt.Println("Player Y ", player.Y())
	fmt.Println("Player Speed ", player.Speed())
}

func TestEventCollectorGeneralJoin(t *testing.T) {
	ec := server.NewEventCollector()

	playerJoinedList := []types.Player{
		{Id: 69, X: 10, Y: 200, Speed: 420},
		{Id: 999, X: 999, Y: 999, Speed: 999},
	}
	flatPlayerJoinedList := utils.NewFlatPlayerJoinedList(flatbuffers.NewBuilder(512), playerJoinedList)

	ec.AddGeneralEvent(utils.NewEventHolder(flatgen.EventKindPlayerJoinedList, flatPlayerJoinedList))
	ec.AddGeneralEvent(utils.NewEventHolder(flatgen.EventKindPlayerJoinedList, flatPlayerJoinedList))

	ec.GetPlayerEventList(2)

	playerEvents, _ := ec.GetPlayerEventList(2)
	fmt.Printf("list.EventsLength(): %v\n", playerEvents.EventsLength())

	rawEvent := &flatgen.RawEvent{}
	playerEvents.Events(rawEvent, 0)

	kindHolder := flatgen.GetRootAsKindHolder(rawEvent.RawDataBytes(), 0)

	fmt.Println(flatgen.EnumNamesEventKind[kindHolder.Kind()])

	playerMoved := flatgen.GetRootAsPlayerJoinedList(rawEvent.RawDataBytes(), 0)

	player := &flatgen.Player{}
	playerMoved.Players(player, 0)

	fmt.Println("Player id ", player.Id())
	fmt.Println("Player X ", player.X())
	fmt.Println("Player Y ", player.Y())
	fmt.Println("Player Speed ", player.Speed())

	playerMoved.Players(player, 1)

	fmt.Println("Player id ", player.Id())
	fmt.Println("Player X ", player.X())
	fmt.Println("Player Y ", player.Y())
	fmt.Println("Player Speed ", player.Speed())
}

func TestBunica(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	flatgen.BunicaEventStart(builder)
	flatgen.BunicaEventAddKind(builder, flatgen.EventKindPlayerHello)
	// flatgen.BunicaEventAddId(builder, 1)
	builder.Finish(flatgen.BunicaEventEnd(builder))
	bunica := builder.FinishedBytes()
	fmt.Println("Bunica", len(bunica))

	fmt.Printf("bunicaEvent: %v\n", len(bunica))
	//
	builder3 := flatbuffers.NewBuilder(1024)

	rawData := builder3.CreateByteVector(bunica)

	flatgen.RawEventStart(builder3)
	flatgen.RawEventAddRawData(builder3, rawData)
	// builder3.Finish()

	// RawEvent := builder3.FinishedBytes()

	// fmt.Printf("len(flatgen.GetRootAsRawEvent(RawEvent, 0).Table().Bytes): %v\n", len(flatgen.GetRootAsRawEvent(RawEvent, 0).Table().Bytes))

	// builder4 := flatbuffers.NewBuilder(1024)
	RawEvent := flatgen.RawEventEnd(builder3)
	// rawEventOffset := builder4.CreateByteVector(RawEvent)

	flatgen.EventListStartEventsVector(builder3, 1)
	builder3.PrependUOffsetT(RawEvent) // This will set them the other way around but order is not a problem
	eventVector := builder3.EndVector(1)

	flatgen.EventListStart(builder3)
	flatgen.EventListAddEvents(builder3, eventVector)
	builder3.Finish(flatgen.EventListEnd(builder3))

	FINAL := builder3.FinishedBytes()
	fmt.Printf("len(FINAL): %v\n", len(FINAL))
}

func TestBunica2(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	flatgen.BunicaEventStart(builder)
	flatgen.BunicaEventAddKind(builder, flatgen.EventKindPlayerMoved)
	// flatgen.BunicaEventAddId(builder, 1)
	builder.Finish(flatgen.BunicaEventEnd(builder))
	bunica := builder.FinishedBytes()
	fmt.Println("Bunica", len(bunica))

	builder2 := flatbuffers.NewBuilder(1024)

	rawData := builder2.CreateByteVector(bunica)

	flatgen.RawEventStart(builder2)
	flatgen.RawEventAddRawData(builder2, rawData)

	RawEvent := flatgen.RawEventEnd(builder2)
	// rawEventOffset := builder4.CreateByteVector(RawEvent)

	flatgen.EventListStartEventsVector(builder2, 1)
	builder2.PrependUOffsetT(RawEvent) // This will set them the other way around but order is not a problem
	eventVector := builder2.EndVector(1)

	flatgen.EventListStart(builder2)
	flatgen.EventListAddEvents(builder2, eventVector)
	builder2.Finish(flatgen.EventListEnd(builder2))

	FINAL := builder2.FinishedBytes()
	fmt.Printf("len(FINAL): %v\n", len(FINAL))

	ev := &flatgen.RawEvent{}
	flatgen.GetRootAsEventList(FINAL, 0).Events(ev, 0)

	fmt.Printf("flatgen.GetRootAsKindHolder(ev.RawDataBytes(), 0).Kind().String(): %v\n", flatgen.GetRootAsKindHolder(ev.RawDataBytes(), 0).Kind().String())
}
