package server

type StatCollector struct {
	tickBuilder *TickStatBuilder

	tickStatList []TickStats

	tickIndex       int
	collectionFrame int
}

func NewStatCollector(collectionFrame int) StatCollector {
	return StatCollector{
		tickBuilder:     &TickStatBuilder{},
		tickStatList:    make([]TickStats, collectionFrame),
		tickIndex:       0,
		collectionFrame: collectionFrame,
	}
}

func (sc *StatCollector) Tick() *TickStatBuilder {
	return sc.tickBuilder
}

func (sc *StatCollector) FinishTick() {
	sc.tickIndex++
	sc.tickStatList = append(sc.tickStatList, sc.tickBuilder.AvgTickStat())
	sc.tickBuilder.Reset()
}

func (sc *StatCollector) AvgStatsIfReady() *AvgStats {
	avgStats := &AvgStats{}

	if sc.tickIndex == sc.collectionFrame {
		for i := range sc.tickStatList {
			avgStats.AvgActivePlayers += sc.tickStatList[i].ActivePlayers
			avgStats.AvgDataSentPerPlayer += sc.tickStatList[i].DataSentPerPlayer
			avgStats.AvgEventsSentPerTick += sc.tickStatList[i].EventsSent
			avgStats.AvgEventsRecvPerTick += sc.tickStatList[i].EventsReceived
			avgStats.AvgTickProcessingTime += sc.tickStatList[i].ProcessingTime
			avgStats.AvgMessageSize += sc.tickStatList[i].AvgMessageSize
			avgStats.MaxMessageSize = max(avgStats.MaxMessageSize, sc.tickStatList[i].MaxMessageSize)
		}

		n := float64(len(sc.tickStatList))

		avgStats.AvgActivePlayers /= n
		avgStats.AvgDataSentPerPlayer /= n
		avgStats.AvgEventsSentPerTick /= n
		avgStats.AvgEventsRecvPerTick /= n
		avgStats.AvgTickProcessingTime /= n
		avgStats.AvgMessageSize /= n

		return avgStats
	}

	return nil
}

func (sc *StatCollector) ResetFrame() {
	sc.tickBuilder.Reset()
	sc.tickStatList = sc.tickStatList[:0]
	sc.tickIndex = 0
}

type TickStatBuilder struct {
	eventsReceivedCount int
	eventsSentCount     int
	totalSentDataSize   int
	processTime         float64
	activePlayers       int

	maxMessageSize int
}

func (tsb *TickStatBuilder) AddEventsReceived(count int) {
	tsb.eventsReceivedCount += count
}

func (tsb *TickStatBuilder) AddEventsSent(count int) {
	tsb.eventsSentCount += count
}

func (tsb *TickStatBuilder) AddMessageSize(size int) {
	tsb.totalSentDataSize += size

	tsb.maxMessageSize = max(tsb.maxMessageSize, size)
}

func (tsb *TickStatBuilder) AddTime(seconds float64) {
	tsb.processTime += seconds
}

func (tsb *TickStatBuilder) AddActivePlayers(count int) {
	tsb.activePlayers += count
}

func (tsb *TickStatBuilder) AvgTickStat() TickStats {
	return TickStats{
		ProcessingTime:    tsb.processTime,
		EventsReceived:    float64(tsb.eventsReceivedCount),
		DataSentPerPlayer: float64(tsb.totalSentDataSize) / float64(max(tsb.activePlayers, 1)),
		TotalDataSent:     float64(tsb.totalSentDataSize),
		AvgMessageSize:    float64(tsb.totalSentDataSize) / float64(max(tsb.eventsSentCount, 1)),
		EventsSent:        float64(tsb.eventsSentCount),
		MaxMessageSize:    float64(tsb.maxMessageSize),
		ActivePlayers:     float64(tsb.activePlayers),
	}
}

func (tsb *TickStatBuilder) Reset() {
	tsb.eventsReceivedCount = 0
	tsb.eventsSentCount = 0
	tsb.totalSentDataSize = 0
	tsb.processTime = 0
	tsb.activePlayers = 0
	tsb.maxMessageSize = 0
}

type AvgStats struct {
	AvgDataSentPerPlayer  float64
	AvgEventsSentPerTick  float64
	AvgEventsRecvPerTick  float64
	AvgTickProcessingTime float64
	AvgActivePlayers      float64
	MaxMessageSize        float64
	AvgMessageSize        float64
}

type TickStats struct {
	EventsReceived    float64
	DataSentPerPlayer float64
	AvgMessageSize    float64
	EventsSent        float64
	ProcessingTime    float64
	MaxMessageSize    float64
	TotalDataSent     float64
	ActivePlayers     float64
}
