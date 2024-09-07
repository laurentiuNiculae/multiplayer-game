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

(() => {
    const conn = new WebSocket("/websocket")
    let myID = undefined
    let Players = {}

    let gameCanvas = document.getElementById("canvas")
    gameCanvas.width = 800
    gameCanvas.height = 800
    let ctx = gameCanvas.getContext("2d")

    conn.addEventListener("open", (event) => {
        console.log("websocket connected")
    })

    conn.addEventListener('close', ev => {
        console.log("websocket disconnected")
    })

    conn.addEventListener("message", (event) => {
        if (myID === undefined) {
            const message = JSON.parse(event.data)
            if (isHello(message)) {
                myID = message.Id
                console.log("We got hello!", `Our id = "${message.Id}"`)
            } else {
                console.log("ERROR: Expected hello message")
            }
        } else {
            const message = JSON.parse(event.data)
            
            switch (true) {
                case isPlayerJoined(message):
                    console.log("New Player Joined", `His id = "${message.Player.Id}"`, message)
                    message.Player.MovingLeft = false
                    message.Player.MovingRight = false
                    message.Player.MovingUp = false
                    message.Player.MovingDown = false
                    
                    Players[message.Player.Id] = message.Player
                    break
                case isPlayerQuit(message):
                    delete Players[message.Id]
                    console.log("New Player Quit", `His id = "${message.Id}"`, message)
                    break
                case isPlayerMoved(message):
                    console.log("WOWOWOWOWOWO PLAYER MOVEDD")
                    const playerId = message.Player.Id
                    let player = Players[playerId]
                    player.MovingLeft = message.MovingLeft
                    player.MovingRight = message.MovingRight
                    player.MovingUp = message.MovingUp
                    player.MovingDown = message.MovingDown

                    Players[playerId] = player
                    break
                default:
                    console.log("bogus amogus", message)
            }
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
            if (player.MovingLeft && player.X-5 >= 0) {
				player.X = player.X-5
			}
			if (player.MovingRight && player.X+5 < WorldWidth - 20) {
				player.X = player.X+5
			}
			if (player.MovingUp && player.Y-5 >= 0) {
				player.Y = player.Y-5
			}
			if (player.MovingDown && player.Y+5 < WorldHeight - 20) {
				player.Y = player.Y+5
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

            conn.send(JSON.stringify({
                Kind: "PlayerMoved",
                Player: {
                    Id: Players[myID].Id,
                    X:Players[myID].X,
                    Y:Players[myID].Y
                },
                MovingUp: Players[myID].MovingUp,
                MovingLeft: Players[myID].MovingLeft,
                MovingDown: Players[myID].MovingDown,
                MovingRight: Players[myID].MovingRight
            }))
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

            conn.send(JSON.stringify({
                Kind: "PlayerMoved",
                Player: {
                    Id: Players[myID].Id,
                    X:Players[myID].X,
                    Y:Players[myID].Y
                },
                MovingUp: Players[myID].MovingUp,
                MovingLeft: Players[myID].MovingLeft,
                MovingDown: Players[myID].MovingDown,
                MovingRight: Players[myID].MovingRight
            }))
        }
    })

    window.requestAnimationFrame((timestamp) => {
        prevTimestamp = timestamp
        window.requestAnimationFrame(frame)
    })
})();