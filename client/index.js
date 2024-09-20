import * as Game from './flatgen/game.js';
import * as flatbuffers from './flatbuffers/flatbuffers.js';
const Port = 6969;
const WorldWidth = 800 * 2;
const WorldHeight = 600 * 2;
function isHello(x) {
    return x && x.Kind === "PlayerHello";
}
function isPlayerJoined(x) {
    return x && x.Kind === "PlayerJoined";
}
function isPlayerJoinedList(x) {
    return x && x.Kind === "PlayerJoinedList";
}
function isPlayerQuit(x) {
    return x && x.Kind === "PlayerQuit";
}
function isPlayerMoved(x) {
    return x && x.Kind === "PlayerMoved";
}
function min(a, b) {
    if (a < b) {
        return a;
    }
    return b;
}
function max(a, b) {
    if (a > b) {
        return a;
    }
    return b;
}
function rawBlobToFlatEvent(rawEventBlob) {
    var array = new Uint8Array(rawEventBlob);
    var buf = new flatbuffers.ByteBuffer(array);
    return Game.Event.getRootAsEvent(buf);
}
function rawBlobToFlatEventList(rawEventBlob) {
    var array = new Uint8Array(rawEventBlob);
    maxMessageSize = max(maxMessageSize, array.length);
    lastMessageSize = array.length;
    var buf = new flatbuffers.ByteBuffer(array);
    return Game.EventList.getRootAsEventList(buf);
}
function getFlatPlayerHello(array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerHello.getRootAsPlayerHello(eventDataBuf);
}
function getFlatPlayerJoined(array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerJoined.getRootAsPlayerJoined(eventDataBuf);
}
function getFlatPlayerJoinedList(array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerJoinedList.getRootAsPlayerJoinedList(eventDataBuf);
}
function getFlatPlayerQuit(array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerQuit.getRootAsPlayerQuit(eventDataBuf);
}
function getFlatPlayerMoved(array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerMoved.getRootAsPlayerMoved(eventDataBuf);
}
function getFlatPlayerMovedList(array) {
    let eventDataBuf = new flatbuffers.ByteBuffer(array);
    return Game.PlayerMovedList.getRootAsPlayerMovedList(eventDataBuf);
}
let maxMessageSize = 0;
let lastMessageSize = 0;
(() => {
    const conn = new WebSocket("/websocket");
    let myID = undefined;
    let Players = new Map();
    let gameCanvas = document.getElementById("canvas");
    gameCanvas.width = WorldWidth;
    gameCanvas.height = WorldHeight;
    let ctx = gameCanvas.getContext("2d");
    conn.addEventListener("open", (event) => {
        console.log("websocket connected");
    });
    conn.addEventListener('close', ev => {
        console.log("websocket disconnected");
    });
    conn.addEventListener("message", (event) => {
        if (myID === undefined) {
            event.data.arrayBuffer().then((rawEventBlob) => {
                let flatEvent = rawBlobToFlatEvent(rawEventBlob);
                let playerHello = getFlatPlayerHello(flatEvent.dataArray());
                myID = playerHello.id();
                console.log("We got hello!", `Our id = "${myID}"`);
                let builder = new flatbuffers.Builder(256);
                let helloResponse = Game.PlayerHelloConfirm.createPlayerHelloConfirm(builder, myID);
                builder.finish(helloResponse);
                let eventData = builder.asUint8Array();
                let flatEventData = builder.createByteVector(eventData);
                let kind = builder.createString("PlayerHelloConfirm");
                let eventResponse = Game.Event.createEvent(builder, kind, flatEventData);
                builder.finish(eventResponse);
                let responseBytes = builder.asUint8Array();
                conn.send(responseBytes);
            });
        }
        else {
            event.data.arrayBuffer().then((rawEventBlob) => {
                let flatEventList = rawBlobToFlatEventList(rawEventBlob);
                console.log(`Received events len=${flatEventList.eventsLength()}`);
                for (let i = 0; i < flatEventList.eventsLength(); i++) {
                    let rawFlatEvent = flatEventList.events(i);
                    let flatEvent = rawBlobToFlatEvent(rawFlatEvent.rawDataArray());
                    switch (flatEvent.kind()) {
                        case "PlayerJoined":
                            let playerJoined = getFlatPlayerJoined(flatEvent.dataArray());
                            // console.log("New Player Joined", `His id = "${playerJoined.player().id()}"`)
                            Players[playerJoined.player().id()] = {
                                Id: playerJoined.player().id(),
                                Speed: playerJoined.player().speed(),
                                X: playerJoined.player().x(),
                                Y: playerJoined.player().y(),
                                MovingLeft: playerJoined.player().movingLeft(),
                                MovingRight: playerJoined.player().movingRight(),
                                MovingUp: playerJoined.player().movingUp(),
                                MovingDown: playerJoined.player().movingDown()
                            };
                            break;
                        case "PlayerJoinedList":
                            let playerJoinedList = getFlatPlayerJoinedList(flatEvent.dataArray());
                            for (let i = 0; i < playerJoinedList.playersLength(); i++) {
                                let playerJoined = playerJoinedList.players(i);
                                // console.log("New Player Joined List", `His id = "${playerJoined.id()}"`)
                                Players[playerJoined.id()] = {
                                    Id: playerJoined.id(),
                                    Speed: playerJoined.speed(),
                                    X: playerJoined.x(),
                                    Y: playerJoined.y(),
                                    MovingLeft: playerJoined.movingLeft(),
                                    MovingRight: playerJoined.movingRight(),
                                    MovingUp: playerJoined.movingUp(),
                                    MovingDown: playerJoined.movingDown()
                                };
                            }
                            break;
                        case "PlayerQuit":
                            let playerQuit = getFlatPlayerQuit(flatEvent.dataArray());
                            delete Players[playerQuit.id()];
                            console.log("New Player Quit", `His id = "${playerQuit.id()}"`);
                            break;
                        case "PlayerMovedList":
                            const playerMovedList = getFlatPlayerMovedList(flatEvent.dataArray());
                            // console.log(`Player Moved Count = ${playerMovedList.playersLength()}`)
                            for (let i = 0; i < playerMovedList.playersLength(); i++) {
                                const playerMoved = playerMovedList.players(i);
                                let player = Players[playerMoved.id()];
                                if (player === undefined) {
                                    player = {};
                                }
                                player.X = playerMoved.x();
                                player.Y = playerMoved.y();
                                player.MovingLeft = playerMoved.movingLeft();
                                player.MovingRight = playerMoved.movingRight();
                                player.MovingUp = playerMoved.movingUp();
                                player.MovingDown = playerMoved.movingDown();
                                Players[playerMoved.id()] = player;
                            }
                            break;
                        default:
                            console.log("bogus amogus", event.data);
                    }
                }
            });
        }
        console.log("MESSAGE SIZE IS ", lastMessageSize / 1024, "KB", "MAX: ", maxMessageSize / 1024, "KB");
    });
    let prevTimestamp = 0;
    let frame = (timestamp) => {
        let delta = (timestamp - prevTimestamp) / 1000;
        prevTimestamp = timestamp;
        ctx.fillStyle = 'white';
        ctx.fillRect(0, 0, ctx.canvas.width, ctx.canvas.height);
        ctx.fillStyle = 'red';
        for (const [id, player] of Object.entries(Players)) {
            let movedDelta = delta * player.Speed;
            if (player.MovingLeft && player.X - movedDelta >= 0) {
                player.X = player.X - movedDelta;
            }
            if (player.MovingRight && player.X + movedDelta < WorldWidth - 20) {
                player.X = player.X + movedDelta;
                // console.log("movedDelta: ", movedDelta)
            }
            if (player.MovingUp && player.Y - movedDelta >= 0) {
                player.Y = player.Y - movedDelta;
                // console.log("movedDelta: ", movedDelta)
            }
            if (player.MovingDown && player.Y + movedDelta < WorldHeight - 20) {
                player.Y = player.Y + movedDelta;
                // console.log("movedDelta: ", movedDelta)
            }
            Players[id] = player;
            ctx.fillRect(player.X, player.Y, 8, 8);
        }
        window.requestAnimationFrame(frame);
    };
    window.addEventListener("keydown", (e) => {
        if (!e.repeat) {
            // console.log("keydown")
            switch (e.code) {
                case "KeyW":
                    {
                        Players[myID].MovingUp = true;
                    }
                    break;
                case "KeyA":
                    {
                        Players[myID].MovingLeft = true;
                    }
                    break;
                case "KeyS":
                    {
                        Players[myID].MovingDown = true;
                    }
                    break;
                case "KeyD":
                    {
                        Players[myID].MovingRight = true;
                    }
                    break;
            }
            let builder = new flatbuffers.Builder(256);
            let player = Players[myID];
            let flatPlayer = Game.Player.createPlayer(builder, myID, player.X, player.Y, player.Speed, player.MovingLeft, player.MovingRight, player.MovingUp, player.MovingDown);
            let playerMoved = Game.PlayerMoved.createPlayerMoved(builder, flatPlayer);
            builder.finish(playerMoved);
            let playerMovedBytes = builder.asUint8Array();
            let kind = builder.createString("PlayerMoved");
            let data = builder.createByteVector(playerMovedBytes);
            let eventResponse = Game.Event.createEvent(builder, kind, data);
            builder.finish(eventResponse);
            let responseBytes = builder.asUint8Array();
            conn.send(responseBytes);
        }
    });
    window.addEventListener("keyup", (e) => {
        if (!e.repeat) {
            // console.log("keyup")
            switch (e.code) {
                case "KeyW":
                    {
                        Players[myID].MovingUp = false;
                    }
                    break;
                case "KeyA":
                    {
                        Players[myID].MovingLeft = false;
                    }
                    break;
                case "KeyS":
                    {
                        Players[myID].MovingDown = false;
                    }
                    break;
                case "KeyD":
                    {
                        Players[myID].MovingRight = false;
                    }
                    break;
            }
            let builder = new flatbuffers.Builder(256);
            let player = Players[myID];
            let flatPlayer = Game.Player.createPlayer(builder, myID, player.X, player.Y, player.Speed, player.MovingLeft, player.MovingRight, player.MovingUp, player.MovingDown);
            let playerMoved = Game.PlayerMoved.createPlayerMoved(builder, flatPlayer);
            builder.finish(playerMoved);
            let playerMovedBytes = builder.asUint8Array();
            let kind = builder.createString("PlayerMoved");
            let data = builder.createByteVector(playerMovedBytes);
            let eventResponse = Game.Event.createEvent(builder, kind, data);
            builder.finish(eventResponse);
            let responseBytes = builder.asUint8Array();
            conn.send(responseBytes);
        }
    });
    window.requestAnimationFrame((timestamp) => {
        prevTimestamp = timestamp;
        window.requestAnimationFrame(frame);
    });
})();
