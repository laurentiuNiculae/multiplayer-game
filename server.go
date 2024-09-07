package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/coder/websocket"
)

var ServerFPT = 30

var idCounter = 0
var WorldWidth = 800
var WorldHeight = 600
var Port = "6969"
var Address = "127.0.0.1:" + Port
var HttpAddress = "http://127.0.0.1:" + Port
var Players = map[int]PlayerWithSocket{}
var EventQueue = make(chan Event, 200)

func Json(x any) []byte {
	bytes, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}

	return bytes
}

func NotifyAll(msg []byte) {
	for _, player := range Players {
		err := player.Conn.Write(context.Background(), websocket.MessageText, msg)
		if err != nil {
			continue
		}
	}
}

func NotifyAllElse(msg []byte, except ...int) {
	for _, player := range Players {
		if !slices.Contains(except, player.Id) {
			err := player.Conn.Write(context.Background(), websocket.MessageText, msg)
			if err != nil {
				continue
			}
		}
	}
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		wcon, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
		}

		idCounter++

		player := Player{
			Id: idCounter,
			X:  rand.Intn(800),
			Y:  rand.Intn(800),
		}

		Players[player.Id] = PlayerWithSocket{
			Conn:   wcon,
			Player: player,
		}

		defer func() {
			wcon.CloseNow()
			delete(Players, player.Id)

			EventQueue <- Event{
				Kind: PlayerQuitKind,
				Conn: wcon,
				Data: PlayerQuit{Kind: PlayerQuitKind, Id: player.Id},
			}

			fmt.Println("Player", player.Id, "diconnected")
		}()

		EventQueue <- Event{
			Kind: PlayerHelloKind,
			Conn: wcon,
			Data: PlayerHello{Kind: PlayerHelloKind, Id: player.Id},
		}

		for {
			_, dataBytes, err := wcon.Read(ctx)
			if err != nil {
				return
			}

			kind, data, err := getMessageKindAndData(dataBytes)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				continue
			}

			EventQueue <- Event{
				Kind: kind,
				Data: data,
				Conn: wcon,
			}
		}
	})

	go func() {
		WaitServerIsReady(HttpAddress)
		fmt.Println("Listening to server")
	}()

	go tick()

	err := http.ListenAndServe(Address, mux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
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

func tick() {
	ticker := time.NewTicker(1 * time.Second / time.Duration(ServerFPT))

	for range ticker.C {
		for i := 0; i < len(EventQueue); i++ {
			event := <-EventQueue

			switch event.Kind {
			case PlayerHelloKind:
				ctx := context.Background()
				playerHello := event.Data.(PlayerHello)

				fmt.Println("Player connected: ", playerHello.Id)

				err := event.Conn.Write(ctx, websocket.MessageText, Json(playerHello))
				if err != nil {
					fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
				}

				currentPlayer := Players[playerHello.Id]
				for _, otherPlayer := range Players {
					otherPlayer.Conn.Write(ctx, websocket.MessageText, Json(PlayerJoined{
						Kind:   PlayerJoinedKind,
						Player: currentPlayer.Player,
					}))

					if otherPlayer.Id != currentPlayer.Id {
						currentPlayer.Conn.Write(ctx, websocket.MessageText, Json(PlayerJoined{
							Kind:   PlayerJoinedKind,
							Player: otherPlayer.Player,
						}))
					}
				}
			case PlayerQuitKind:
				playerQuit := event.Data.(PlayerQuit)

				NotifyAll(Json(playerQuit))
			case PlayerMovedKind:
				playerMoved := event.Data.(PlayerMoved)

				player := Players[playerMoved.Player.Id]
				player.MovingLeft = playerMoved.MovingLeft
				player.MovingRight = playerMoved.MovingRight
				player.MovingUp = playerMoved.MovingUp
				player.MovingDown = playerMoved.MovingDown

				Players[playerMoved.Player.Id] = player

				NotifyAll(Json(playerMoved))
			}
		}

		for i, player := range Players {
			if player.MovingLeft && player.X-5 >= 0 {
				player.X = player.X - 5
			}
			if player.MovingRight && player.X+5 < WorldWidth {
				player.X = player.X + 5
			}
			if player.MovingUp && player.Y-5 >= 0 {
				player.Y = player.Y - 5
			}
			if player.MovingDown && player.Y+5 < WorldHeight {
				player.Y = player.Y + 5
			}

			Players[i] = player
		}

		// fmt.Println("tick")
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
