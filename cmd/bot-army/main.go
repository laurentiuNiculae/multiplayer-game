package main

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	. "test/pkg/types"
	"test/pkg/types/utils"
	"time"

	flatgen "test/pkg/types/flatgen/game"

	"github.com/coder/websocket"
	flatbuffers "github.com/google/flatbuffers/go"
)

func GetMoveUpEvent(builder *flatbuffers.Builder, player Player) *flatgen.Event {
	player.MovingUp = true
	player.MovingDown = false
	player.MovingLeft = false
	player.MovingRight = false

	playerMovedBytes := utils.NewFlatPlayerMoved(builder, player).Table().Bytes

	return utils.NewFlatEvent(builder, PlayerMovedKind, playerMovedBytes)
}
func GetMoveDownEvent(builder *flatbuffers.Builder, player Player) *flatgen.Event {
	player.MovingUp = false
	player.MovingDown = true
	player.MovingLeft = false
	player.MovingRight = false

	playerMovedBytes := utils.NewFlatPlayerMoved(builder, player).Table().Bytes

	return utils.NewFlatEvent(builder, PlayerMovedKind, playerMovedBytes)
}
func GetMoveLeftEvent(builder *flatbuffers.Builder, player Player) *flatgen.Event {
	player.MovingUp = false
	player.MovingDown = false
	player.MovingLeft = true
	player.MovingRight = false

	playerMovedBytes := utils.NewFlatPlayerMoved(builder, player).Table().Bytes

	return utils.NewFlatEvent(builder, PlayerMovedKind, playerMovedBytes)
}
func GetMoveRightEvent(builder *flatbuffers.Builder, player Player) *flatgen.Event {
	player.MovingUp = false
	player.MovingDown = false
	player.MovingLeft = false
	player.MovingRight = true

	playerMovedBytes := utils.NewFlatPlayerMoved(builder, player).Table().Bytes

	return utils.NewFlatEvent(builder, PlayerMovedKind, playerMovedBytes)
}

var ServerFPT = 30
var WorldWidth = float64(800 * 2)
var WorldHeight = float64(600 * 2)

func GameLoop(ctx context.Context, conn *websocket.Conn, playerUpdateChan <-chan Player) {
	myPlayer := <-playerUpdateChan

	moveTicker := time.NewTicker(600 * time.Millisecond)
	moveCount := 0

	ticker := time.NewTicker(1 * time.Second / time.Duration(ServerFPT))
	previousTime, delta := time.Now(), time.Duration(0)

	<-ticker.C

	for range ticker.C {
		select {
		case <-ticker.C:
			select {
			case <-ctx.Done():
				return
			case player := <-playerUpdateChan:
				myPlayer = player
			default:
				break
			}

			delta, previousTime = time.Since(previousTime), time.Now()

			movedDelta := delta.Seconds() * myPlayer.Speed

			if myPlayer.MovingLeft && myPlayer.X-movedDelta >= 0 {
				myPlayer.X = myPlayer.X - movedDelta
			}
			if myPlayer.MovingRight && myPlayer.X+movedDelta < WorldWidth {
				myPlayer.X = myPlayer.X + movedDelta
			}
			if myPlayer.MovingUp && myPlayer.Y-movedDelta >= 0 {
				myPlayer.Y = myPlayer.Y - movedDelta
			}
			if myPlayer.MovingDown && myPlayer.Y+movedDelta < WorldHeight {
				myPlayer.Y = myPlayer.Y + movedDelta
			}
		case <-moveTicker.C:
			builder := flatbuffers.NewBuilder(256)

			switch moveCount {
			case 0:
				err := conn.Write(ctx, websocket.MessageBinary, GetMoveUpEvent(builder, myPlayer).Table().Bytes)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
				}
			case 1:
				err := conn.Write(ctx, websocket.MessageBinary, GetMoveRightEvent(builder, myPlayer).Table().Bytes)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
				}
			case 2:
				err := conn.Write(ctx, websocket.MessageBinary, GetMoveDownEvent(builder, myPlayer).Table().Bytes)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
				}
			case 3:
				err := conn.Write(ctx, websocket.MessageBinary, GetMoveLeftEvent(builder, myPlayer).Table().Bytes)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
				}
			}

			moveCount = (moveCount + 1) % 4
		}
	}
}

func RunBot(ctx context.Context, wg *sync.WaitGroup, Id int) {
	defer func() {
		fmt.Printf("Bot%v shutting down. ZzzZzz\n", Id)
		wg.Done()
	}()

	conn, _, err := websocket.Dial(ctx, "http://localhost:6969/websocket", nil)
	if err != nil {
		fmt.Printf("Bot%v error: %s\n", Id, err)
	}

	playerUpdateChan := make(chan Player)

	var myId int

	builder := flatbuffers.NewBuilder(256)

	{
		_, bytes, err := conn.Read(ctx)
		if err != nil {
			fmt.Printf("Bot%v error: %s\n", Id, err)
		}

		_, data, err := utils.ParseEventBytes(bytes)
		if err != nil {
			fmt.Printf("Bot%v error: %s\n", Id, err)
		}
		playerHello := data.(*flatgen.PlayerHello)

		myId = int(playerHello.Id())
		fmt.Printf("Bot%v Got Id: '%v'\n", Id, myId)

		// Confirm the hello message
		{
			playerHelloConfirm := utils.NewFlatPlayerHelloConfirm(builder, myId)
			helloConfirmEvent := utils.NewFlatEvent(builder, PlayerHelloConfirmKind, playerHelloConfirm.Table().Bytes)

			err := conn.Write(ctx, websocket.MessageBinary, helloConfirmEvent.Table().Bytes)
			if err != nil {
				fmt.Println(err)
			}
		}

		go GameLoop(ctx, conn, playerUpdateChan)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_, dataBytes, err := conn.Read(ctx)
			if err != nil {
				fmt.Printf("Bot%v error: %s\n", Id, err)
				return
			}

			kind, data, err := utils.ParseEventBytes(dataBytes)
			if err != nil {
				fmt.Printf("Bot%v error: %s\n", Id, err)
				continue
			}

			if kind == PlayerJoinedKind {
				playerJoined := data.(*flatgen.PlayerJoined)
				player := &flatgen.Player{}
				if playerJoined.Player(player).Id() == int32(myId) {
					playerUpdateChan <- Player{
						Id:    int(player.Id()),
						X:     float64(player.X()),
						Y:     float64(player.Y()),
						Speed: float64(player.Speed()),
					}

					fmt.Printf("Bot%v Confirmed Join: \n", Id)
				}

			} else if kind == PlayerMovedKind {
				playerMoved := data.(*flatgen.PlayerMoved)
				player := &flatgen.Player{}
				if playerMoved.Player(player).Id() == int32(myId) {
					playerUpdateChan <- Player{
						Id:    int(player.Id()),
						X:     float64(player.X()),
						Y:     float64(player.Y()),
						Speed: float64(player.Speed()),
					}
				}
			}
		}
	}
}

func main() {
	NumBots := 7

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(NumBots)

	for ID := range NumBots {
		time.Sleep(time.Millisecond * 500)
		go RunBot(ctx, &wg, ID)
	}

	<-ctx.Done()

	fmt.Println("Finishing execution")

	wg.Wait()
}
