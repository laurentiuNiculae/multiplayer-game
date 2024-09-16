package main

import (
	"context"
	"fmt"
	. "test/pkg/types"
	"test/pkg/types/utils"

	flatgen "test/pkg/types/flatgen/game"

	"github.com/coder/websocket"
	flatbuffers "github.com/google/flatbuffers/go"
)

func main() {
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "http://localhost:6969/websocket", nil)
	if err != nil {
		fmt.Println(err)
	}

	var myId int
	builder := flatbuffers.NewBuilder(256)

	{
		_, bytes, err := conn.Read(ctx)
		if err != nil {
			fmt.Println(err)
		}

		_, data, err := utils.ParseEventBytes(bytes)
		if err != nil {
			fmt.Println(err)
		}
		playerHello := data.(*flatgen.PlayerHello)

		myId = int(playerHello.Id())

		// Confirm the hello message
		{
			playerHelloConfirm := utils.NewFlatPlayerHelloConfirm(builder, myId)
			helloConfirmEvent := utils.NewFlatEvent(builder, PlayerHelloConfirmKind, playerHelloConfirm.Table().Bytes)

			err := conn.Write(ctx, websocket.MessageBinary, helloConfirmEvent.Table().Bytes)
			if err != nil {
				fmt.Println(err)
			}
		}

		for {
			_, dataBytes, err := conn.Read(ctx)
			if err != nil {
				return
			}

			kind, data, err := utils.ParseEventBytes(dataBytes)
			if err != nil {
				// game.log.Errorf("err: %v\n", err)
				continue
			}

			fmt.Println(kind)
			fmt.Println(data)
		}
	}
}
