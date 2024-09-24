package server

import (
	"fmt"

	"github.com/laurentiuNiculae/multiplayer-game/pkg/types"
	flatgen "github.com/laurentiuNiculae/multiplayer-game/pkg/types/flatgen/game"

	flatbuffers "github.com/google/flatbuffers/go"
)

type EventListBuilder struct {
	builder        *flatbuffers.Builder
	events         []types.EventHolder
	builderStarted bool
}

func (elb *EventListBuilder) AddRawEvent(event types.EventHolder) error {
	if elb.builderStarted {
		return fmt.Errorf("another build is already started, need to reset first")
	}

	elb.events = append(elb.events, event)
	return nil
}

func (elb *EventListBuilder) GetFlatEventList(generalEvents []types.EventHolder) (*flatgen.EventList, int) {
	totalEventCount := len(elb.events) + len(generalEvents)
	if totalEventCount == 0 {
		return nil, 0
	}

	elb.builderStarted = true

	rawEventList := make([]flatbuffers.UOffsetT, totalEventCount)
	for i := 0; i < len(elb.events); i++ {
		rawDataOffset := elb.builder.CreateByteVector(elb.events[i].Bytes())

		flatgen.RawEventStart(elb.builder)
		flatgen.RawEventAddRawData(elb.builder, rawDataOffset)
		rawEventList[i] = flatgen.RawEventEnd(elb.builder)
	}

	for i := len(elb.events); i < totalEventCount; i++ {
		j := i - len(elb.events)
		rawDataOffset := elb.builder.CreateByteVector(generalEvents[j].Bytes())

		flatgen.RawEventStart(elb.builder)
		flatgen.RawEventAddRawData(elb.builder, rawDataOffset)
		rawEventList[i] = flatgen.RawEventEnd(elb.builder)
	}

	flatgen.EventListStartEventsVector(elb.builder, totalEventCount)
	for i := range rawEventList {
		elb.builder.PrependUOffsetT(rawEventList[i]) // This will set them the other way around but order is not a problem
	}

	eventList := elb.builder.EndVector(totalEventCount)

	flatgen.EventListStart(elb.builder)
	flatgen.EventListAddEvents(elb.builder, eventList)
	elb.builder.Finish(flatgen.EventListEnd(elb.builder))

	return flatgen.GetRootAsEventList(elb.builder.FinishedBytes(), 0), totalEventCount
}

type EventCollector struct {
	playerEvents  map[int]EventListBuilder
	generalEvents []types.EventHolder
}

func NewEventCollector() *EventCollector {
	return &EventCollector{
		playerEvents: map[int]EventListBuilder{},
	}
}

func (es *EventCollector) AddEvent(playerId int, event types.EventHolder) {
	if event.Kind() == flatgen.EventKindNilEvent {
		return
	}

	playerEventsBuilder, ok := es.playerEvents[playerId]
	if !ok {
		playerEventsBuilder = EventListBuilder{
			builder: flatbuffers.NewBuilder(256),
		}
	}

	playerEventsBuilder.AddRawEvent(event)
	es.playerEvents[playerId] = playerEventsBuilder
}

func (es *EventCollector) AddGeneralEvent(event types.EventHolder) {
	es.generalEvents = append(es.generalEvents, event)
}

func (es *EventCollector) GetPlayerEventList(playerId int) (*flatgen.EventList, int) {
	playerEventsBuilder, ok := es.playerEvents[playerId]
	if !ok {
		playerEventsBuilder = EventListBuilder{builder: flatbuffers.NewBuilder(512)} // TODO: Is this ok?
	}

	return playerEventsBuilder.GetFlatEventList(es.generalEvents)
}

func (es *EventCollector) Reset() {
	for id, eventListBuilder := range es.playerEvents {
		eventListBuilder.builder.Reset()
		eventListBuilder.events = eventListBuilder.events[:0]

		es.playerEvents[id] = eventListBuilder
	}

	clear(es.generalEvents)
	es.generalEvents = es.generalEvents[:0]
}

func (es *EventCollector) RemovePlayer(playerId int) {
	delete(es.playerEvents, playerId)
}
