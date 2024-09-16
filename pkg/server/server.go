package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"time"

	"test/pkg/log"
	. "test/pkg/types"
	flatgen "test/pkg/types/flatgen/game"
	"test/pkg/types/utils"

	"github.com/coder/websocket"
	flatbuffers "github.com/google/flatbuffers/go"
)

var ServerFPT = 30
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
	Players     PlayerStore
	EventQueue  chan Event
	IdGenerator IdGenerator
	mux         *http.ServeMux
	log         log.MeloLog
}

func NewGame() GameServer {
	return GameServer{
		Players:     NewPlayerStore(),
		EventQueue:  make(chan Event, 200),
		IdGenerator: IdGenerator{},
		mux:         http.NewServeMux(),
		log:         log.New(os.Stdout),
	}
}

func (game *GameServer) Start() {
	game.mux.Handle("/", http.FileServer(http.Dir(".")))
	game.mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		wcon, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
		}

		playerId := game.IdGenerator.NewId()

		defer func() {
			wcon.CloseNow()
			game.Players.Delete(playerId)
			builder := flatbuffers.NewBuilder(128)

			game.EventQueue <- Event{
				PlayerId: playerId,
				Kind:     PlayerQuitKind,
				Conn:     wcon,
				Data:     flatgen.GetRootAsPlayerQuit(GetFlatPlayerQuit(builder, playerId), 0),
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
		WaitServerIsReady(HttpAddress)
		game.log.Info("Listening to server")
	}()

	go game.Tick()

	err := http.ListenAndServe(Address, game.mux)
	if err != nil {
		game.log.Errorf("err: %s\n", err.Error())
	}
}

func (game *GameServer) Tick() {
	ticker := time.NewTicker(1 * time.Second / time.Duration(ServerFPT))
	previousTime, delta := time.Now(), time.Duration(0)

	<-ticker.C

	for range ticker.C {
		// start := time.Now()
		for range len(game.EventQueue) {
			ctx := context.Background()
			event := <-game.EventQueue

			switch event.Kind {
			case PlayerHelloKind:
				playerHello := event.Data.(PlayerHello)

				newPlayer := PlayerWithSocket{
					Conn: event.Conn,
					Player: Player{
						Id:    playerHello.Id,
						Speed: rand.Float64()*800 + 200,
						X:     rand.Float64()*float64(WorldWidth)/2 + float64(WorldWidth)/4,
						Y:     rand.Float64()*float64(WorldHeight)/2 + +float64(WorldHeight)/4,
					},
				}

				game.Players.Set(newPlayer.Id, newPlayer)

				game.log.Infof("Player connected: '%v'", playerHello.Id)

				builder := flatbuffers.NewBuilder(1024)

				eventData := GetFlatPlayerHello(builder, newPlayer.Player)
				eventBytes := GetFlatEvent(builder, PlayerHelloKind, eventData)

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

				builder := flatbuffers.NewBuilder(1024)
				newPlayer, _ := game.Players.Get(event.PlayerId)

				flatNewPlayerJoined := GetFlatPlayerJoined(builder, newPlayer.Player)
				flatNewPlayerJoinedEvent := GetFlatEvent(builder, PlayerJoinedKind, flatNewPlayerJoined)

				for _, otherPlayer := range game.Players.All() {
					otherPlayer.Conn.Write(ctx, websocket.MessageBinary, flatNewPlayerJoinedEvent)

					flatOtherPlayerJoined := GetFlatPlayerJoined(builder, otherPlayer.Player)
					flatOtherPlayerJoinedEvent := GetFlatEvent(builder, PlayerJoinedKind, flatOtherPlayerJoined)
					if otherPlayer.Id != newPlayer.Id {
						newPlayer.Conn.Write(ctx, websocket.MessageBinary, flatOtherPlayerJoinedEvent)
					}
				}
			case PlayerQuitKind:
				playerQuit := event.Data.(*flatgen.PlayerQuit)

				if playerQuit.Id() != int32(event.PlayerId) {
					event.Conn.CloseNow()
					game.log.Errorf("player '%s' tried to cheat", event.PlayerId)
				}

				response := GetFlatEvent(flatbuffers.NewBuilder(256), PlayerQuitKind,
					playerQuit.Table().Bytes)

				game.NotifyAll(response)
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

				response := GetFlatEvent(flatbuffers.NewBuilder(256), PlayerMovedKind,
					playerMoved.Table().Bytes)

				game.NotifyAll(response)
			}
		}

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

		// game.log.Debugf("%06f", time.Since(start).Seconds())
		// game.log.Info(delta.String())
	}
}

func GetFlatPlayerHello(builder *flatbuffers.Builder, newPlayer Player) []byte {
	flatgen.PlayerHelloStart(builder)
	flatgen.PlayerHelloAddId(builder, int32(newPlayer.Id))
	flatgen.FinishPlayerHelloBuffer(builder, flatgen.PlayerHelloEnd(builder))

	return builder.FinishedBytes()
}

func GetFlatPlayerQuit(builder *flatbuffers.Builder, playerId int) []byte {
	flatgen.PlayerQuitStart(builder)
	flatgen.PlayerQuitAddId(builder, int32(playerId))
	flatgen.FinishPlayerQuitBuffer(builder, flatgen.PlayerQuitEnd(builder))

	return builder.FinishedBytes()
}

func GetFlatPlayerJoined(builder *flatbuffers.Builder, newPlayer Player) []byte {
	flatPlayer := GetFlatPlayer(builder, newPlayer)

	flatgen.PlayerJoinedStart(builder)
	flatgen.PlayerJoinedAddPlayer(builder, flatPlayer)
	flatgen.FinishPlayerJoinedBuffer(builder, flatgen.PlayerJoinedEnd(builder))

	return builder.FinishedBytes()
}

func GetFlatPlayer(builder *flatbuffers.Builder, newPlayer Player) flatbuffers.UOffsetT {
	flatgen.PlayerStart(builder)
	flatgen.PlayerAddId(builder, int32(newPlayer.Id))
	flatgen.PlayerAddX(builder, int32(newPlayer.X))
	flatgen.PlayerAddY(builder, int32(newPlayer.Y))
	flatgen.PlayerAddSpeed(builder, int32(newPlayer.Speed))
	flatgen.PlayerAddMovingDown(builder, newPlayer.MovingDown)
	flatgen.PlayerAddMovingLeft(builder, newPlayer.MovingLeft)
	flatgen.PlayerAddMovingRight(builder, newPlayer.MovingRight)
	flatgen.PlayerAddMovingUp(builder, newPlayer.MovingUp)

	return flatgen.PlayerEnd(builder)
}

// func GetTestEvent(builder *flatbuffers.Builder) []byte {
// 	flatId := builder.CreateString("BUNICA_ID")

// 	flatgen.BunicaEventStart(builder)
// 	flatgen.BunicaEventAddId(builder, flatId)
// 	builder.Finish(flatgen.BunicaEventEnd(builder))
// 	return builder.FinishedBytes()
// }

func GetFlatEvent(builder *flatbuffers.Builder, kind string, bytes []byte) []byte {
	flatKind := builder.CreateByteString([]byte(kind))
	flatBytes := builder.CreateByteVector(bytes)

	flatgen.EventStart(builder)
	flatgen.EventAddKind(builder, flatKind)
	flatgen.EventAddData(builder, flatBytes)
	builder.Finish(flatgen.EventEnd(builder))

	return builder.FinishedBytes()
}

// func getMessageKindAndData(data []byte) (string, any, error) {
// 	kindHolder := KindHolder{}

// 	if err := json.Unmarshal(data, &kindHolder); err != nil {
// 		return "", nil, err
// 	}

// 	switch kindHolder.Kind {
// 	case PlayerQuitKind:
// 		var playerQuit PlayerQuit

// 		if err := json.Unmarshal(data, &playerQuit); err != nil {
// 			return "", nil, err
// 		}

// 		return kindHolder.Kind, playerQuit, nil
// 	case PlayerHelloKind:
// 		var playerHello PlayerHello

// 		if err := json.Unmarshal(data, &playerHello); err != nil {
// 			return "", nil, err
// 		}

// 		return kindHolder.Kind, playerHello, nil
// 	case PlayerJoinedKind:
// 		return PlayerJoinedKind, PlayerJoined{}, fmt.Errorf("server doesn't accept playerJoined messages")
// 	case PlayerMovedKind:
// 		var playerMoved PlayerMoved

// 		if err := json.Unmarshal(data, &playerMoved); err != nil {
// 			return "", nil, err
// 		}

// 		return PlayerMovedKind, playerMoved, nil
// 	default:
// 		return "", nil, fmt.Errorf("ERROR: bogus-amogus kind '%s'", kindHolder.Kind)
// 	}
// }

func (game *GameServer) NotifyAll(msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

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

func Json(x any) []byte {
	bytes, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}

	return bytes
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
