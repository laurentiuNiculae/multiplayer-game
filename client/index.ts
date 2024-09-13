import * as Game from './flatgen/game.js'
import * as flatbuffers from './flatbuffers/flatbuffers.js'

const Port = 6969;
const WorldWidth = 800;
const WorldHeight = 600;

function isHello(x) {
    return x && x.Kind === "PlayerHello"
} 

function isPlayerJoined(x) {
    return x && x.Kind === "PlayerJoined"
}

function isPlayerQuit(x) {
    return x && x.Kind === "PlayerQuit"
}

function isPlayerMoved(x) {
    return x && x.Kind === "PlayerMoved"
}

function min(a, b) {
    if (a < b) {
        return a
    }

    return b
}

function max(a, b) {
    if (a < b) {
        return a
    }

    return b
}

interface Player {
    Speed: number,
    X: number,
    Y: number,
    MovingLeft: boolean,
    MovingRight: boolean,
    MovingUp: boolean,
    MovingDown: boolean,
}

function rawBlobToFlatEvent(rawEventBlob) {
    var array = new Uint8Array(rawEventBlob)
    var buf = new flatbuffers.ByteBuffer(array);
    return Game.Event.getRootAsEvent(buf);
}

function getFlatPlayerHello(array: Uint8Array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerHello.getRootAsPlayerHello(eventDataBuf);
}

function getFlatPlayerJoined(array: Uint8Array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerJoined.getRootAsPlayerJoined(eventDataBuf);
}

function getFlatPlayerQuit(array: Uint8Array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerQuit.getRootAsPlayerQuit(eventDataBuf);
}

function getFlatPlayerMoved(array: Uint8Array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerMoved.getRootAsPlayerMoved(eventDataBuf);
}

(() => {
    const conn = new WebSocket("/websocket")
    let myID = undefined
    let bunica = false
    let Players = new Map<Number, Player>()

    let gameCanvas = document.getElementById("canvas") as HTMLCanvasElement

    gameCanvas.width = WorldWidth
    gameCanvas.height = WorldHeight
    let ctx = gameCanvas.getContext("2d")

    conn.addEventListener("open", (event) => {
        console.log("websocket connected")
    })

    conn.addEventListener('close', ev => {
        console.log("websocket disconnected")
    })

    conn.addEventListener("message", (event) => {
        if (myID === undefined) {
            event.data.arrayBuffer().then((rawEventBlob) => {
                let flatEvent = rawBlobToFlatEvent(rawEventBlob)
                let playerHello = getFlatPlayerHello(flatEvent.dataArray())

                myID = playerHello.id()
                
                let builder = new flatbuffers.Builder(256)
                let helloResponse = Game.PlayerHelloConfirm.createPlayerHelloConfirm(builder, myID)
                let kind = builder.createString("PlayerHelloConfirm")
                let eventResponse = Game.Event.createEvent(builder, kind, helloResponse)
                builder.finish(eventResponse)
                let responseBytes = builder.asUint8Array()

                conn.send(responseBytes)

                console.log("We got hello!", `Our id = "${myID}"`)
            })
        }  else {
            event.data.arrayBuffer().then((rawEventBlob) => {
                let flatEvent = rawBlobToFlatEvent(rawEventBlob)
            
                switch (flatEvent.kind()) {
                    case "PlayerJoined":
                        let playerJoined = getFlatPlayerJoined(flatEvent.dataArray())
    
                        console.log("New Player Joined", `His id = "${playerJoined.player().id()}"`)
                        
                        Players[playerJoined.player().id()] = {
                            Id: playerJoined.player().id(),
                            Speed: playerJoined.player().speed(),
                            X: playerJoined.player().x(),
                            Y: playerJoined.player().y(),
                            MovingLeft:  playerJoined.player().movingLeft(),
                            MovingRight:  playerJoined.player().movingRight(),
                            MovingUp:  playerJoined.player().movingUp(),
                            MovingDown:  playerJoined.player().movingDown()
                        }
                        break
                    case "PlayerQuit":
                        let playerQuit = getFlatPlayerQuit(flatEvent.dataArray())

                        delete Players[playerQuit.id()]
                        console.log("New Player Quit", `His id = "${playerQuit.id()}"`)
                        break
                    case "PlayerMoved":
                        const playerMoved = getFlatPlayerMoved(flatEvent.dataArray())
                        const playerId = playerMoved.player().id()

                        let player = Players[playerId]
                        player.X = playerMoved.player().x()
                        player.Y = playerMoved.player().y()
                        player.MovingLeft = playerMoved.player().movingLeft()
                        player.MovingRight = playerMoved.player().movingRight()
                        player.MovingUp = playerMoved.player().movingUp()
                        player.MovingDown = playerMoved.player().movingDown()
    
                        Players[playerId] = player
                        break
                    default:
                        console.log("bogus amogus", event.data)
                }
            })
        }
    })

    let prevTimestamp = 0

    let frame = (timestamp) => {
        let delta = (timestamp - prevTimestamp)/1000
        prevTimestamp = timestamp

        ctx.fillStyle = 'white'
        ctx.fillRect(0, 0, ctx.canvas.width, ctx.canvas.height)
        ctx.fillStyle = 'red'

        
        for (const [id, player] of Object.entries(Players)) {
            let movedDelta = delta * player.Speed

            if (player.MovingLeft && player.X-movedDelta >= 0) {
                player.X = player.X-movedDelta
                // console.log("movedDelta: ", movedDelta)
                console.log("speed: ", player.Speed)
			}
			if (player.MovingRight && player.X+movedDelta < WorldWidth - 20) {
				player.X = player.X+movedDelta
                // console.log("movedDelta: ", movedDelta)
			}
			if (player.MovingUp && player.Y-movedDelta >= 0) {
				player.Y = player.Y-movedDelta
                // console.log("movedDelta: ", movedDelta)
			}
			if (player.MovingDown && player.Y+movedDelta < WorldHeight - 20) {
				player.Y = player.Y+movedDelta
                // console.log("movedDelta: ", movedDelta)
			}

            Players[id] = player

            ctx.fillRect(player.X, player.Y, 20, 20)
        }

        window.requestAnimationFrame(frame)
    }

    window.addEventListener("keypress", (e) => {
        if (!e.repeat) {
            console.log("keydown")
            switch (e.code) {
                case "KeyW": {Players[myID].MovingUp = true} break;
                case "KeyA": {Players[myID].MovingLeft = true} break;
                case "KeyS": {Players[myID].MovingDown = true} break;
                case "KeyD": {Players[myID].MovingRight = true} break;
            }

            let builder = new flatbuffers.Builder(256)
            let player = Players[myID] as Player
            let flatPlayer = Game.Player.createPlayer(builder, myID, player.X, player.Y,
                player.Speed, player.MovingLeft, player.MovingRight, player.MovingUp, player.MovingDown
            )
            let playerMoved = Game.PlayerMoved.createPlayerMoved(builder, flatPlayer)
            builder.finish(playerMoved)
            let playerMovedBytes = builder.asUint8Array()

            let kind = builder.createString("PlayerMoved")
            let data = builder.createByteVector(playerMovedBytes)

            let eventResponse = Game.Event.createEvent(builder, kind, data)
            builder.finish(eventResponse)
            let responseBytes = builder.asUint8Array()

            conn.send(responseBytes)
        }
    })

    window.addEventListener("keyup", (e) => {
        if (!e.repeat) {
            console.log("keyup")
            switch (e.code) {
                case "KeyW": {Players[myID].MovingUp = false} break;
                case "KeyA": {Players[myID].MovingLeft = false} break;
                case "KeyS": {Players[myID].MovingDown = false} break;
                case "KeyD": {Players[myID].MovingRight = false} break;
            }

            let builder = new flatbuffers.Builder(256)
            let player = Players[myID] as Player
            let flatPlayer = Game.Player.createPlayer(builder, myID, player.X, player.Y,
                player.Speed, player.MovingLeft, player.MovingRight, player.MovingUp, player.MovingDown
            )
            let playerMoved = Game.PlayerMoved.createPlayerMoved(builder, flatPlayer)
            builder.finish(playerMoved)
            let playerMovedBytes = builder.asUint8Array()

            let kind = builder.createString("PlayerMoved")
            let data = builder.createByteVector(playerMovedBytes)

            let eventResponse = Game.Event.createEvent(builder, kind, data)
            builder.finish(eventResponse)
            let responseBytes = builder.asUint8Array()

            conn.send(responseBytes)
        }
    })

    window.requestAnimationFrame((timestamp) => {
        prevTimestamp = timestamp
        window.requestAnimationFrame(frame)
    })
})();