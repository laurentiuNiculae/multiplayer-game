package server

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"slices"
	"sort"
	"time"

	"test/pkg/log"
	. "test/pkg/types"
	flatgen "test/pkg/types/flatgen/game"
	"test/pkg/types/utils"

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
	mux            *http.ServeMux
	log            log.MeloLog
}

func NewGame() GameServer {
	return GameServer{
		Players:        NewPlayerStore(),
		EventQueue:     make(chan Event, 2000),
		IdGenerator:    IdGenerator{},
		EventCollector: NewEventCollector(),
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
				Kind:     PlayerQuitKind,
				Conn:     wcon,
				Data:     flatgen.GetRootAsPlayerQuit(utils.GetFlatPlayerQuit(builder, playerId), 0),
			}

			game.log.Infof("Player '%v' diconnected", playerId)
		}()

		game.EventQueue <- Event{
			PlayerId: playerId,
			Kind:     PlayerHelloKind,
			Conn:     wcon,
			Data:     PlayerHello{Kind: PlayerHelloKind, Id: playerId},
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

func PrintMemUsage(log log.MeloLog) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Debugf("Alloc = %v MiB\n\tTotalAlloc = %v MiB\n\tSys = %v MiB\n", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys))
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func (game *GameServer) Tick() {
	utils.WaitServerIsReady(HttpAddress)

	tickTimeArr := make([]float64, 30)
	timeI := 0

	ticker := time.NewTicker(1 * time.Second / time.Duration(ServerFPS))
	previousTime, delta := time.Now(), time.Duration(0)

	playerMovedBuilder := flatbuffers.NewBuilder(512)
	playerMovedList := []*flatgen.PlayerMoved{}

	playerJoinedBuilder := flatbuffers.NewBuilder(512)
	playerJoinedBuilder2 := flatbuffers.NewBuilder(512)
	playerJoinedList := []Player{}

	<-ticker.C

	avgEventsPerTick := float64(0)
	avgDataPerSecond := float64(0)
	totalEvents := float64(0)
	totalEventsPerSecond := float64(0)

	start := time.Now()

	for range ticker.C {
		startTick := time.Now()
		ctx := context.Background()

		eventsPerTick := float64(len(game.EventQueue))

		for range len(game.EventQueue) {
			event := <-game.EventQueue

			switch event.Kind {
			case PlayerHelloKind:
				playerHello := event.Data.(PlayerHello)

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

				game.log.Infof("Player connected: '%v'", playerHello.Id)

				builder := flatbuffers.NewBuilder(512)

				eventData := utils.GetFlatPlayerHello(builder, newPlayer.Player)
				eventBytes := utils.GetFlatEvent(builder, PlayerHelloKind, eventData).Table().Bytes

				err := newPlayer.Conn.Write(ctx, websocket.MessageBinary, eventBytes)
				if err != nil {
					game.log.Errorf("err: %s\n", err.Error())
				}
			case PlayerHelloConfirmKind:
				helloResponse := event.Data.(*flatgen.PlayerHelloConfirm)

				if helloResponse.Id() == int32(event.PlayerId) {
					game.log.Debug("HELLO CONFIRMED BY PLAYER")
				} else {
					game.log.Debugf("player ID doesn't match expected:'%d', given:'%d'", event.PlayerId, helloResponse.Id())
				}

				newPlayer, _ := game.Players.Get(event.PlayerId)

				playerJoinedList = append(playerJoinedList, newPlayer.Player)

				playerJoinedList = append(playerJoinedList, newPlayer.Player)

				for _, otherPlayer := range game.Players.All() {
					builder2 := flatbuffers.NewBuilder(512)

					flatOtherPlayerJoined := utils.GetFlatPlayerJoined(builder2, otherPlayer.Player)
					flatOtherPlayerJoinedEvent := utils.GetFlatEvent(builder2, PlayerJoinedKind, flatOtherPlayerJoined)
					if otherPlayer.Id != newPlayer.Id {
						game.EventCollector.AddEvent(newPlayer.Id, flatOtherPlayerJoinedEvent)
					}
				}
			case PlayerQuitKind:
				playerQuit := event.Data.(*flatgen.PlayerQuit)

				if playerQuit.Id() != int32(event.PlayerId) {
					event.Conn.CloseNow()
					game.log.Errorf("player '%s' tried to cheat", event.PlayerId)
				}

				playerQuitEvent := utils.GetFlatEvent(flatbuffers.NewBuilder(512), PlayerQuitKind,
					playerQuit.Table().Bytes)

				game.Players.Delete(event.PlayerId)
				game.EventCollector.RemovePlayer(event.PlayerId)

				for _, player := range game.Players.All() {
					game.EventCollector.AddEvent(player.Id, playerQuitEvent)
				}

			case PlayerMovedKind:
				playerMoved := event.Data.(*flatgen.PlayerMoved)
				newPlayerInfo := playerMoved.Player(nil)

				if newPlayerInfo.Id() != int32(event.PlayerId) {
					event.Conn.CloseNow()
					game.log.Errorf("player '%s' tried to cheat", event.PlayerId)
				}

				player, _ := game.Players.Get(int(newPlayerInfo.Id())) // TODO _
				player.MovingLeft = newPlayerInfo.MovingLeft()
				player.MovingRight = newPlayerInfo.MovingRight()
				player.MovingUp = newPlayerInfo.MovingUp()
				player.MovingDown = newPlayerInfo.MovingDown()

				playerMoved.Player(nil).MutateX(int32(player.X))
				playerMoved.Player(nil).MutateY(int32(player.Y))

				game.Players.Set(int(newPlayerInfo.Id()), player)

				// playerMovedEvent := utils.GetFlatEvent(flatbuffers.NewBuilder(512), PlayerMovedKind,
				// 	playerMoved.Table().Bytes)

				playerMovedList = append(playerMovedList, playerMoved)

				// for _, player := range game.Players.All() {
				// 	game.EventCollector.AddEvent(player.Id, playerMovedEvent)
				// }
			}
		}

		// calculate all players that moved event and send it.
		if len(playerMovedList) > 0 {
			flatPlayerMovedList := utils.NewFlatPlayerMovedList(playerMovedBuilder, playerMovedList)
			game.EventCollector.AddGeneralEvent(utils.NewFlatEvent(playerMovedBuilder, PlayerMovedListKind,
				flatPlayerMovedList.Table().Bytes))
		}

		if len(playerJoinedList) > 0 {
			flatPlayerJoinedList := utils.NewFlatPlayerJoinedList(playerJoinedBuilder2, playerJoinedList)
			game.EventCollector.AddGeneralEvent(utils.NewFlatEvent(playerJoinedBuilder2, PlayerJoinedListKind,
				flatPlayerJoinedList.Table().Bytes))
		}

		// collect events here then send them.
		for id, player := range game.Players.All() {
			eventList := game.EventCollector.GetPlayerEventList(id)

			if eventList != nil {
				avgDataPerSecond += float64(len(eventList.Table().Bytes))
				player.Conn.Write(ctx, websocket.MessageBinary, eventList.Table().Bytes)
			}
		}

		game.EventCollector.Reset()
		clear(playerMovedList)

		playerMovedList = playerMovedList[:0]
		playerMovedBuilder.Reset()

		playerJoinedList = playerJoinedList[:0]
		playerJoinedBuilder.Reset()
		playerJoinedBuilder2.Reset()

		delta, previousTime = time.Since(previousTime), time.Now()

		for i, player := range game.Players.All() {
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

		if timeI == 29 {
			sort.Slice(tickTimeArr, func(i, j int) bool {
				return tickTimeArr[i] < tickTimeArr[j]
			})

			game.log.Debugf("Tick: %06f  Avg-Events: %02f TotalEvents/s: %02f avgDataPerSecond: %02f", tickTimeArr[30/2], avgEventsPerTick, totalEventsPerSecond, avgDataPerSecond/1024/1024/time.Since(start).Seconds())
			PrintMemUsage(game.log)

			timeI = 0
			avgEventsPerTick = 0
		}

		tickTimeArr[timeI] = time.Since(startTick).Seconds()
		avgEventsPerTick += eventsPerTick / 30
		totalEvents += eventsPerTick
		totalEventsPerSecond = totalEvents / time.Since(start).Seconds()
		timeI++
	}
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
