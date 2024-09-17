package server

import (
	"fmt"
	flatgen "test/pkg/types/flatgen/game"

	flatbuffers "github.com/google/flatbuffers/go"
)

type EventListBuilder struct {
	builder        *flatbuffers.Builder
	events         []*flatgen.Event
	builderStarted bool
}

func (elb *EventListBuilder) AddEvent(event *flatgen.Event) error {
	if elb.builderStarted {
		return fmt.Errorf("another build is already started, need to reset first")
	}

	elb.events = append(elb.events, event)
	return nil
}

func (elb *EventListBuilder) GetFlatEventList(generalEvents []*flatgen.Event) *flatgen.EventList {
	totalEventCount := len(elb.events) + len(generalEvents)
	if totalEventCount == 0 {
		return nil
	}

	elb.builderStarted = true

	rawEventList := make([]flatbuffers.UOffsetT, totalEventCount)
	for i := 0; i < len(elb.events); i++ {
		if len(elb.events[i].Kind()) == 0 {
			fmt.Println("XD")
		}
		rawDataOffset := elb.builder.CreateByteVector(elb.events[i].Table().Bytes)

		flatgen.RawEventStart(elb.builder)
		flatgen.RawEventAddRawData(elb.builder, rawDataOffset)
		rawEventList[i] = flatgen.RawEventEnd(elb.builder)
	}

	for i := len(elb.events); i < totalEventCount; i++ {
		j := i - len(elb.events)
		rawDataOffset := elb.builder.CreateByteVector(generalEvents[j].Table().Bytes)

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

	return flatgen.GetRootAsEventList(elb.builder.FinishedBytes(), 0)
}

type EventCollector struct {
	playerEvents  map[int]EventListBuilder
	generalEvents []*flatgen.Event
}

func NewEventCollector() *EventCollector {
	return &EventCollector{
		playerEvents: map[int]EventListBuilder{},
	}
}

func (es *EventCollector) AddEvent(playerId int, event *flatgen.Event) {
	playerEventsBuilder, ok := es.playerEvents[playerId]
	if !ok {
		playerEventsBuilder = EventListBuilder{
			builder: flatbuffers.NewBuilder(256),
		}
	}

	playerEventsBuilder.AddEvent(event)
	es.playerEvents[playerId] = playerEventsBuilder
}

func (es *EventCollector) AddGeneralEvent(event *flatgen.Event) {
	es.generalEvents = append(es.generalEvents, event)
}

func (es *EventCollector) GetPlayerEventList(playerId int) *flatgen.EventList {
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
