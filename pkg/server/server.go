package server

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"slices"
	"time"

	"github.com/laurentiuNiculae/multiplayer-game/pkg/log"
	. "github.com/laurentiuNiculae/multiplayer-game/pkg/types"
	flatgen "github.com/laurentiuNiculae/multiplayer-game/pkg/types/flatgen/game"
	"github.com/laurentiuNiculae/multiplayer-game/pkg/types/utils"

	"github.com/coder/websocket"
	flatbuffers "github.com/google/flatbuffers/go"
)

var ServerFPS = 30
var WorldWidth = float64(800 * 2)
var WorldHeight = float64(600 * 2)
var Port = "6969"
var Address = "127.0.0.1:" + Port
var HttpAddress = "http://127.0.0.1:" + Port

type IdGenerator struct {
	idCounter int
}

func (igen *IdGenerator) NewId() int {
	igen.idCounter++

	return igen.idCounter
}

type GameServer struct {
	Players        PlayerStore
	EventQueue     chan Event
	IdGenerator    IdGenerator
	EventCollector *EventCollector
	StatCollector  *StatCollector
	FlatCache      *FlatCache
	mux            *http.ServeMux
	log            log.MeloLog
}

func NewGame() GameServer {
	return GameServer{
		Players:        NewPlayerStore(),
		EventQueue:     make(chan Event, 2000),
		IdGenerator:    IdGenerator{},
		EventCollector: NewEventCollector(),
		StatCollector:  NewStatCollector(ServerFPS),
		FlatCache:      NewFlatCache(),
		mux:            http.NewServeMux(),
		log:            log.New(os.Stdout),
	}
}

func (game *GameServer) Start(ctx context.Context) {
	game.mux.Handle("/", http.FileServer(http.Dir(".")))
	game.mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		wcon, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
		}

		playerId := game.IdGenerator.NewId()

		defer func() {
			builder := flatbuffers.NewBuilder(128)

			game.EventQueue <- Event{
				PlayerId: playerId,
				Kind:     flatgen.EventKindPlayerQuit,
				Conn:     wcon,
				Data:     flatgen.GetRootAsPlayerQuit(utils.NewFlatPlayerQuit(builder, playerId).Table().Bytes, 0),
			}

			// game.log.Infof("Player '%v' diconnected", playerId)
		}()

		game.EventQueue <- Event{
			PlayerId: playerId,
			Kind:     flatgen.EventKindPlayerHello,
			Conn:     wcon,
			Data:     PlayerHello{Kind: flatgen.EventKindPlayerHello, Id: playerId},
		}

		for {
			_, dataBytes, err := wcon.Read(ctx)
			if err != nil {
				return
			}

			kind, data, err := utils.ParseEventBytes(dataBytes)
			if err != nil {
				game.log.Errorf("err: %v\n", err)
				continue
			}

			game.EventQueue <- Event{
				PlayerId: playerId,
				Kind:     kind,
				Data:     data,
				Conn:     wcon,
			}
		}
	})

	go func() {
		utils.WaitServerIsReady(HttpAddress)
		game.log.Info("Listening to server")
	}()

	go game.Tick()

	go func() {
		err := http.ListenAndServe(Address, game.mux)
		if err != nil {
			game.log.Errorf("err: %s\n", err.Error())
		}
	}()

	<-ctx.Done()
}

func (game *GameServer) Tick() {
	utils.WaitServerIsReady(HttpAddress)

	ticker := time.NewTicker(1 * time.Second / time.Duration(ServerFPS))
	previousTime, delta := time.Now(), time.Duration(0)

	bufferPool := NewBuilderPool(512, 4)

	playerMovedList := []*flatgen.PlayerMoved{}
	playerJoinedList := []Player{}

	<-ticker.C

	for range ticker.C {
		startTick := time.Now()
		ctx := context.Background()

		game.StatCollector.Tick().AddEventsReceived(len(game.EventQueue))

		for range len(game.EventQueue) {
			event := <-game.EventQueue

			switch event.Kind {
			case flatgen.EventKindPlayerHello:
				playerHello := event.Data.(PlayerHello)

				if playerHello.Id != event.PlayerId {
					event.Conn.CloseNow()
					game.log.Errorf("player '%s' tried to cheat", event.PlayerId)
				}

				newPlayer := PlayerWithSocket{
					Conn: event.Conn,
					Player: Player{
						Id:    playerHello.Id,
						Speed: rand.Float64()*100 + 200,
						X:     rand.Float64()*float64(WorldWidth)/4 + float64(WorldWidth)/2,
						Y:     rand.Float64()*float64(WorldHeight)/4 + +float64(WorldHeight)/2,
					},
				}

				game.Players.Set(newPlayer.Id, newPlayer)

				// game.log.Infof("Player connected: '%v'", playerHello.Id)

				eventData := utils.NewFlatPlayerHello(bufferPool.GetFreeBuilder(), newPlayer.Player).Table().Bytes

				err := newPlayer.Conn.Write(ctx, websocket.MessageBinary, eventData)
				if err != nil {
					game.log.Errorf("err: %s\n", err.Error())
				}
			case flatgen.EventKindPlayerHelloConfirm:
				helloResponse := event.Data.(*flatgen.PlayerHelloConfirm)

				if helloResponse.Id() != int32(event.PlayerId) {
					game.log.Debugf("player ID doesn't match expected:'%d', given:'%d'", event.PlayerId, helloResponse.Id())
					event.Conn.CloseNow()
				}

				newPlayer, _ := game.Players.Get(event.PlayerId)

				playerJoinedList = append(playerJoinedList, newPlayer.Player)

				playerJoinedList = append(playerJoinedList, newPlayer.Player)

				for _, otherPlayer := range game.Players.All() {
					otherPlayerJoined := game.FlatCache.GetMutatedPlayerJoined(otherPlayer.Id, otherPlayer.Player)

					flatOtherPlayerJoinedEvent := utils.NewEventHolder(flatgen.EventKindPlayerJoined, otherPlayerJoined)
					if otherPlayer.Id != newPlayer.Id {
						game.EventCollector.AddEvent(newPlayer.Id, flatOtherPlayerJoinedEvent)
					}
				}
			case flatgen.EventKindPlayerQuit:
				playerQuit := event.Data.(*flatgen.PlayerQuit)

				if playerQuit.Id() != int32(event.PlayerId) {
					event.Conn.CloseNow()
					game.log.Errorf("player '%s' tried to cheat", event.PlayerId)
				}

				playerQuitEvent := utils.NewEventHolder(flatgen.EventKindPlayerQuit, playerQuit)

				game.Players.Delete(event.PlayerId)
				game.EventCollector.RemovePlayer(event.PlayerId)
				game.FlatCache.RemoveJoin(event.PlayerId)

				for _, player := range game.Players.All() {
					game.EventCollector.AddEvent(player.Id, playerQuitEvent)
				}
			case flatgen.EventKindPlayerMoved:
				playerMoved := event.Data.(*flatgen.PlayerMoved)
				newPlayerInfo := playerMoved.Player(nil)

				if newPlayerInfo.Id() != int32(event.PlayerId) {
					event.Conn.Close(websocket.StatusNormalClosure, "Cheating!")
					game.log.Errorf("player '%v' tried to cheat, expected id '%v' but got '%v'", event.PlayerId, event.PlayerId, newPlayerInfo.Id())
				}

				player, _ := game.Players.Get(int(newPlayerInfo.Id())) // TODO _
				player.MovingLeft = newPlayerInfo.MovingLeft()
				player.MovingRight = newPlayerInfo.MovingRight()
				player.MovingUp = newPlayerInfo.MovingUp()
				player.MovingDown = newPlayerInfo.MovingDown()

				playerMoved.Player(nil).MutateX(int32(player.X))
				playerMoved.Player(nil).MutateY(int32(player.Y))

				game.Players.Set(int(newPlayerInfo.Id()), player)
				playerMovedList = append(playerMovedList, playerMoved)
			}
		}

		// TODO: move this into a EventCollector
		// calculate all players that moved event and send it.
		if len(playerMovedList) > 0 {
			flatPlayerMovedList := utils.NewFlatPlayerMovedList(bufferPool.GetFreeBuilder(), playerMovedList)
			game.EventCollector.AddGeneralEvent(utils.NewEventHolder(flatgen.EventKindPlayerMovedList, flatPlayerMovedList))
		}

		if len(playerJoinedList) > 0 {
			flatPlayerJoinedList := utils.NewFlatPlayerJoinedList(bufferPool.GetFreeBuilder(), playerJoinedList)
			game.EventCollector.AddGeneralEvent(utils.NewEventHolder(flatgen.EventKindPlayerJoinedList, flatPlayerJoinedList))
		}

		// collect events here then send them.
		for id, player := range game.Players.All() {
			eventList, _ := game.EventCollector.GetPlayerEventList(id)

			if eventList != nil {
				game.StatCollector.Tick().AddEventsSent(1)
				game.StatCollector.Tick().AddMessageSize(len(eventList.Table().Bytes))

				err := writeTo(ctx, player, eventList.Table().Bytes)
				if err != nil {
					game.log.Error(err.Error())
				}
			}

			game.StatCollector.Tick().AddActivePlayers(1)
		}

		// TODO: Something to manage state, buffers and stuff like that.
		game.EventCollector.Reset()
		clear(playerMovedList)

		playerMovedList = playerMovedList[:0]
		playerJoinedList = playerJoinedList[:0]

		bufferPool.Reset()

		delta, previousTime = time.Since(previousTime), time.Now()

		for i, player := range game.Players.All() {
			// TODO: UpdateGameState()
			movedDelta := delta.Seconds() * player.Speed

			if player.MovingLeft && player.X-movedDelta >= 0 {
				player.X = player.X - movedDelta
			}
			if player.MovingRight && player.X+movedDelta < WorldWidth {
				player.X = player.X + movedDelta
			}
			if player.MovingUp && player.Y-movedDelta >= 0 {
				player.Y = player.Y - movedDelta
			}
			if player.MovingDown && player.Y+movedDelta < WorldHeight {
				player.Y = player.Y + movedDelta
			}

			game.Players.Set(i, player)
		}

		game.StatCollector.Tick().AddTime(time.Since(startTick).Seconds())
		game.StatCollector.FinishTick()

		if stats := game.StatCollector.AvgStatsIfReady(); stats != nil {
			game.log.Debugf("Tick: %06f  Avg-Events: %.3f AvgDataSentPerPlayer: %.3f KB",
				stats.AvgTickProcessingTime,
				stats.AvgEventsRecvPerTick,
				stats.AvgDataSentPerPlayer/1024,
			)
			PrintMemUsage(game.log)
			game.StatCollector.ResetFrame()
		}
	}
}

func writeTo(ctx context.Context, player PlayerWithSocket, b []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("lost connection with player '%v'", player.Id)
		}
	}()

	return player.Conn.Write(ctx, websocket.MessageBinary, b)
}

func (game *GameServer) NotifyAll(msg []byte) {
	for _, player := range game.Players.All() {
		err := player.Conn.Write(context.Background(), websocket.MessageBinary, msg)
		if err != nil {
			continue
		}
	}
}

func (game *GameServer) NotifyAllElse(msg []byte, except ...int) {
	for _, player := range game.Players.All() {
		if !slices.Contains(except, player.Id) {
			err := player.Conn.Write(context.Background(), websocket.MessageBinary, msg)
			if err != nil {
				continue
			}
		}
	}
}

func PrintMemUsage(log log.MeloLog) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Debugf("Alloc = %v MiB\n\tTotalAlloc = %v MiB\n\tSys = %v MiB\n", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys))
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
