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

	"github.com/coder/websocket"
)

var ServerFPT = 30
var WorldWidth = float64(800)
var WorldHeight = float64(600)
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

			game.EventQueue <- Event{
				Kind: PlayerQuitKind,
				Conn: wcon,
				Data: PlayerQuit{Kind: PlayerQuitKind, Id: playerId},
			}

			game.log.Infof("Player '%v' diconnected", playerId)
		}()

		game.EventQueue <- Event{
			Kind: PlayerHelloKind,
			Conn: wcon,
			Data: PlayerHello{Kind: PlayerHelloKind, Id: playerId},
		}

		for {
			_, dataBytes, err := wcon.Read(ctx)
			if err != nil {
				return
			}

			kind, data, err := getMessageKindAndData(dataBytes)
			if err != nil {
				game.log.Errorf("err: %v\n", err)
				continue
			}

			game.EventQueue <- Event{
				Kind: kind,
				Data: data,
				Conn: wcon,
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
		for i := 0; i < len(game.EventQueue); i++ {
			event := <-game.EventQueue

			switch event.Kind {
			case PlayerHelloKind:
				ctx := context.Background()
				playerHello := event.Data.(PlayerHello)
				newPlayerConn := event.Conn

				newPlayer := Player{
					Id:    playerHello.Id,
					Speed: rand.Float64()*800 + 200,
					X:     rand.Float64() * float64(WorldWidth),
					Y:     rand.Float64() * float64(WorldHeight),
				}

				// registerPlayer
				game.Players.Set(newPlayer.Id, PlayerWithSocket{
					Conn:   event.Conn,
					Player: newPlayer,
				})

				game.log.Infof("Player connected: '%v'", playerHello.Id)

				err := newPlayerConn.Write(ctx, websocket.MessageText, Json(playerHello))
				if err != nil {
					game.log.Errorf("err: %s\n", err.Error())
				}

				for _, otherPlayer := range game.Players.All() {
					otherPlayer.Conn.Write(ctx, websocket.MessageText, Json(PlayerJoined{
						Kind:   PlayerJoinedKind,
						Player: newPlayer,
					}))

					if otherPlayer.Id != newPlayer.Id {
						newPlayerConn.Write(ctx, websocket.MessageText, Json(PlayerJoined{
							Kind:   PlayerJoinedKind,
							Player: otherPlayer.Player,
						}))
					}
				}
			case PlayerQuitKind:
				playerQuit := event.Data.(PlayerQuit)

				game.NotifyAll(Json(playerQuit))
			case PlayerMovedKind:
				playerMoved := event.Data.(PlayerMoved)

				player, ok := game.Players.Get(playerMoved.Player.Id)
				_ = ok // TODO
				player.MovingLeft = playerMoved.MovingLeft
				player.MovingRight = playerMoved.MovingRight
				player.MovingUp = playerMoved.MovingUp
				player.MovingDown = playerMoved.MovingDown

				game.Players.Set(playerMoved.Player.Id, player)
				// collect this information and only then update the clients

				game.NotifyAll(Json(playerMoved))
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

		// game.log.Info(delta.String())
	}
}

func getMessageKindAndData(data []byte) (string, any, error) {
	kindHolder := KindHolder{}

	if err := json.Unmarshal(data, &kindHolder); err != nil {
		return "", nil, err
	}

	switch kindHolder.Kind {
	case PlayerQuitKind:
		var playerQuit PlayerQuit

		if err := json.Unmarshal(data, &playerQuit); err != nil {
			return "", nil, err
		}

		return kindHolder.Kind, playerQuit, nil
	case PlayerHelloKind:
		var playerHello PlayerHello

		if err := json.Unmarshal(data, &playerHello); err != nil {
			return "", nil, err
		}

		return kindHolder.Kind, playerHello, nil
	case PlayerJoinedKind:
		return PlayerJoinedKind, PlayerJoined{}, fmt.Errorf("server doesn't accept playerJoined messages")
	case PlayerMovedKind:
		var playerMoved PlayerMoved

		if err := json.Unmarshal(data, &playerMoved); err != nil {
			return "", nil, err
		}

		return PlayerMovedKind, playerMoved, nil
	default:
		return "", nil, fmt.Errorf("ERROR: bogus-amogus kind '%s'", kindHolder.Kind)
	}
}

func (game *GameServer) NotifyAll(msg []byte) {
	for _, player := range game.Players.All() {
		err := player.Conn.Write(context.Background(), websocket.MessageText, msg)
		if err != nil {
			continue
		}
	}
}

func (game *GameServer) NotifyAllElse(msg []byte, except ...int) {
	for _, player := range game.Players.All() {
		if !slices.Contains(except, player.Id) {
			err := player.Conn.Write(context.Background(), websocket.MessageText, msg)
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
