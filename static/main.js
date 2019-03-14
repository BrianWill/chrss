var canvas = document.getElementById('board');
var ctx = canvas.getContext('2d');
var scoreboard = document.getElementById('scoreboard');
var scoreboardCtx = scoreboard.getContext('2d');

const fanfare = document.getElementById('fanfare');
const campanas = document.getElementById('campanas');
const sword = document.getElementById('sword');
fanfare.volume = 0.5;
campanas.volume = 0.5;
sword.volume = 0.5;

var cardList = document.getElementById('card_list');
var cardDescription = document.getElementById('card_description');
var statusInfo = document.getElementById('status_info');
var passButton = document.getElementById('pass_button');
var waitOpponent = document.getElementById('wait_opponent');
var timer = document.getElementById('timer');
var log = document.getElementById('log');
var logBox = document.getElementById('log_box');
var readyup = document.getElementById('readyup');
var readyupButton = document.querySelector('#readyup > button');

var matchState;

const NO_SELECTED_CARD = -1;
const board = {
    width: 650,
    height: 650,
    nRows: 6,
    nColumns: 6,
};
board.squareHeight = board.height / board.nRows;
board.squareWidth = board.width / board.nColumns;
Object.freeze(board);


// need to keep synced with consts in server code
const highlightOff = 0;
const highlightOn = 1;
const highlightDim = 2;


const mainPhase = 'main';
const kingPlacementPhase = 'kingPlacement';

var piecesImg = new Image();
piecesImg.pieceHeight = 45;
piecesImg.pieceWidth = 45;
piecesImg.src = "/static/pieces.svg";
piecesImg.pieceImageCoords = {
    'white_King': {x: 0, y: 0},
    'white_Queen': {x: 45, y: 0},
    'white_Bishop' : {x: 90, y: 0},
    'white_Knight' : {x: 135, y: 0},
    'white_Rook' : {x: 180, y: 0},
    'white_Pawn' : {x: 225, y: 0},
    'black_King': {x: 0, y: 45},
    'black_Queen': {x: 45, y: 45},
    'black_Bishop' : {x: 90, y: 45},
    'black_Knight' : {x: 135, y: 45},
    'black_Rook' : {x: 180, y: 45},
    'black_Pawn' : {x: 225, y: 45},
};

piecesImg.onload = function (evt) {
    if (matchState) {
        draw(matchState);
    }
};

var skullImg = new Image();
skullImg.src = '/static/skull-red.svg';
skullImg.spriteWidth = 512;
skullImg.spriteHeight = 512;
skullImg.onload = function (evt) {
    if (matchState) {
        draw(matchState);
    }
};

var downArrowImg = new Image();
downArrowImg.src = '/static/arrow_down_red.svg';
downArrowImg.spriteWidth = 512;
downArrowImg.spriteHeight = 512;
downArrowImg.onload = function (evt) {
    if (matchState) {
        draw(matchState);
    }
};


var upArrowImg = new Image();
upArrowImg.src = '/static/arrow_up_green.svg';
upArrowImg.spriteWidth = 512;
upArrowImg.spriteHeight = 512;
upArrowImg.onload = function (evt) {
    if (matchState) {
        draw(matchState);
    }
};


var jesterBlack = new Image();
jesterBlack.src = '/static/black_jester.svg';
jesterBlack.spriteWidth = 512;
jesterBlack.spriteHeight = 512;
jesterBlack.onload = function (evt) {
    if (matchState) {
        draw(matchState);
    }
};

var jesterWhite = new Image();
jesterWhite.src = '/static/white_jester.svg';
jesterWhite.spriteWidth = 512;
jesterWhite.spriteHeight = 512;
jesterWhite.onload = function (evt) {
    if (matchState) {
        draw(matchState);
    }
};

var cardTypes = {
    "vassal": "V",
    "soldier": "S",
    "command": "C",
};

var cardDescriptions = {
    'Rook': `<h3>Rook: 0 rank, 20 HP, 6 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks up/down/left/right. You only get one Rook in the match. When reclaimed, its HP and status effects persist, and you get a Rook card back in your hand. When reclaimed, healed for 5 HP.</div>`,
    'Bishop': `<h3>Bishop: 0 rank, 25 HP, 4 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks diagonally. You only get one Bishop in the match. When reclaimed, its HP and status effects persist, and you get a Bishop card back in your hand.</div>`,
    'Knight': `<h3>Knight: 0 rank, 25 HP, 5 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks are not blocked by other units. Attacks in 'L' shape: two spaces in cardinal direction and one space over. You only get one Knight in the match. When reclaimed, its HP and status effects persist, and you get a Knight card back in your hand.</div>`,
    'Pawn': `<h3>Pawn: 0 rank, 5 HP, 2 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks one space diagonally towards opponent side.</div>`,
    'Queen': `<h3>Queen: 5 rank, 15 HP, 6 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks diagonally and up/down/left/right.</div>`,
    'Castle': `<h3>Castle: 2 rank</h3>
<div>Click either King.<br/><br/>Swaps the clicked King's position with the Rook of the same color. (Can only use Castle on a King whose Rook is on the board.)</div>`,
    'Reclaim Vassal': `<h3>Reclaim Vassal: 2 rank</h3>
<div>Click a Knight, Bishop, or Rook.<br/><br/>The clicked vassal is reclaimed immediately.</div>`,
    'Swap Front Lines': `<h3>Swap Front Lines: 2 rank</h3>
<div>Click a King.<br/><br/>Swaps all pieces between the front and middle rows on the clicked King's side.</div>`,
    'Remove Pawn': `<h3>Remove Pawn: 2 rank</h3>
<div>Click a Pawn of either color to remove.</div>`,
    'Force Combat': `<h3>Force Combat: 3 rank</h3>
<div>Click your King to confirm.<br/><br/>Immediately advances match to combat and the end of round.</div>`,
    'Mirror': `<h3>Mirror: 2 rank</h3>
<div>Click either King.<br/><br/>Moves all pieces of clicked color to their horizontally-mirrored positions, <em>e.g.</em> all pieces in the leftmost column move to the rightmost column and <em>vice versa</em>.</div>`,
    'Heal': `<h3>Heal: 2 rank</h3>
<div>Click any of your pieces (except your King).<br/><br/>Adds 5 HP to a non-King piece (not capped by the piece's starting health).</div>`,
    'Toggle Pawn': `<h3>Toggle Pawn: 2 rank</h3>
<div>Click a Pawn.<br/><br/>Moves a Pawn in the front row to the middle row or moves a Pawn in the middle row to the front row. The destination square must be unoccupied.</div>`,
    'Nuke': `<h3>Nuke: 2 rank</h3>
<div>Click a King.<br/><br/>Immediately inflict 6 damage on all pieces within 1 square of the clicked King and 3 damage on all pieces withing 2 squares of the clicked King.</div>`,
    'Shove': `<h3>Shove: 2 rank</h3>
<div>Click a piece.<br/><br/>Moves a white piece one square towards white's back row; moves a black piece one square towards black's back row.</div>`,
    'Advance': `<h3>Advance: 2 rank</h3>
<div>Click a piece.<br/><br/>Moves a white piece one square towards black's back row; moves a black piece one square towards white's back row.</div>`,
    'Summon Pawn': `<h3>Summon Pawn: 2 rank</h3>
<div>Click your King.<br/><br/>Summons an additional pawn (subject to usual max of 5 pawns and restrictions on pawn placement).</div>`,
    'Jester': `<h3>Jester: 3 rank, 12 HP, 0 attack</h3>
<div>Click free square on your side to place.<br/><br/>Does not attack. Puts 'distract' effect on all adjacent squares except those behind the jester. A piece in a square with distract does not attack.</div>`,
    'Vulnerability': `<h3>Vulnerability: 2 rank, 1 round duration</h3>
<div>Click enemy piece.<br/><br/>Doubles damage the targeted piece takes.</div>`,
    'Amplify': `<h3>Amplify: 2 rank, 1 round duration</h3>
<div>Click allied piece.<br/><br/>Doubles damage the targeted piece inflicts.</div>`,
    'Enrage': `<h3>Enrage: 2 rank, 1 round duration</h3>
<div>Click enemy piece.<br/><br/>Enraged piece hits allies as well as enemeies.</div>`,
    'Dodge': `<h3>Dodge: 2 rank</h3>
<div>Click ally piece that is under threat (going to be hit in combat) and has at least one free adjacent square.<br/><br/>Moves piece to random adjacent free square. (May move piece into enemy territory.)</div>`,
    'Resurrect Vassal': `<h3>Resurrect Vassal: 2 rank</h3>
<div>Click ally king.<br/><br/>Resurrects your dead vassal (knight, king, or bishop) with 5 hp and no status effects.</div>`,
    'Stun Vassal': `<h3>Stun Vassal: 2 rank</h3>
<div>Click enemy vassal.<br/><br/>For 1 round, vassal is DamageImmune, Distracted (does not attack), and Unrelcaimable.</div>`,
    'Transparency': `<h3>Transparency: 2 rank</h3>
<div>Click enemy piece.<br/><br/>For 1 round, piece is Transparent (affected by attacks but does not block them).</div>`,
    'Armor': `<h3>Armor: 2 rank</h3>
<div>Click ally piece other than king.<br/><br/>Adds two points of armor to the piece. Each point of armor negates a point of incoming damage from each attacking piece. Can be stacked and can be removed by Dispell.</div>`,
    'Dispell': `<h3>Dispell: 2 rank</h3>
<div>Click piece.<br/><br/>Removes all status effects (positive and negative) from the piece.</div>`,
    'Poison': `<h3>Poison: 2 rank</h3>
<div>Click enemy piece other than King.<br/><br/>Damages piece every combat phase for 2 HP (unless piece is Damage Immune). Can be stacked and can be removed by Dispell. Vulnerability affects the poison damage. Reclaimed vassals are not damaged by poison while off the board.</div>`,
};


var matchId = window.location.pathname.substring(7);
var url = 'wss://chrss-game.herokuapp.com/ws/' + matchId;   
if (location.hostname == 'localhost') {
    url = 'ws://localhost:5000/ws/' + matchId;  // can't do wss over localhost it seems?
}
var conn = new WebSocket(url);


var waitingResponse = false;

conn.onmessage = function(msg){
    console.log(" <== " + new Date() + " <== \n");
    console.log(msg);

    var response = JSON.parse(msg.data);
    if (response === "ping") {
        return;
    }
    matchState = response;
    if (matchState.error) {
        alert(matchState.error);  // todo: use overlay instead of alert
    }

    if (matchState.color === 'black') {
        matchState.public = matchState.blackPublic;
    } else {
        matchState.public = matchState.whitePublic;
    }
    let kingPos = matchState.private.kingPos;
    if (kingPos) {
        let idx = kingPos.x + kingPos.y * board.nColumns;
        matchState.board[idx] = matchState.public.king;
    }
    if (!matchState.log) {
        matchState.log = [];
    }
    setTimers(matchState);
    draw(matchState);

    updateSquareInfoBox(boardClientX, boardClientY);

    // sounds
    try {
        if (matchState.newRound) {
            fanfare.play();
        } else if (matchState.newTurn) {
            switch (matchState.phase) {
                case 'kingPlacement':
                    sword.play();
                    break;
                case 'main':
                    campanas.play();
                    break;
            }
        }
    } catch (ex) {
        console.log(ex);
    }

    waitingResponse = false;
}

conn.onerror = function(err) {
    console.log("Connection error " + new Date() + " error: ", err);
    console.log(err);
}

conn.onclose = function(err) {
    console.log("Connection close " + new Date(), err);
    readyup.style.display = 'block';
    readyup.innerHTML = '<div>CONNECTION LOST.<br/>DID YOU JOIN THIS MATCH IN ANOTHER BROWSER TAB?<br/>REFRESH TO RECONNECT</div>';
}

conn.onopen = function(){
    conn.send("get_state ");
}

// account for pixel ratio (avoids blurry text on high dpi screens)
if (window.devicePixelRatio) {
    let width = scoreboard.getAttribute('width');
    let height = scoreboard.getAttribute('height');
    scoreboard.setAttribute('width', width * window.devicePixelRatio);
    scoreboard.setAttribute('height', height * window.devicePixelRatio);
    scoreboard.style.width = width + 'px';
    scoreboard.style.height = height + 'px';
    scoreboardCtx.scale(window.devicePixelRatio, window.devicePixelRatio);               

    width = canvas.getAttribute('width');
    height = canvas.getAttribute('height');
    canvas.setAttribute('width', width * window.devicePixelRatio);
    canvas.setAttribute('height', height * window.devicePixelRatio);
    canvas.style.width = width + 'px';
    canvas.style.height = height + 'px';
    ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
}


//  *** draw ***

function draw(matchState) {
    drawBoard(ctx);
    drawPieces(ctx, matchState);
    drawStatusIcons(ctx, matchState);
    drawSquareHighlight(ctx, matchState);
    drawWait(ctx, matchState);
    drawWinner(ctx, matchState.winner);
    drawCards(matchState);
    drawScoreboard(scoreboardCtx, matchState);
    drawButtons(matchState);
    drawTimer(matchState);
    drawReadyUp(matchState);
    drawLog(matchState);

    function drawReadyUp(matchState) {
        if (matchState.phase === 'readyUp') {
            if (matchState.public.ready) {
                readyup.innerHTML = '<div>WAITING FOR OTHER PLAYER TO READY UP</div>';
            }
            readyup.style.display = 'block';
        } else {
            readyup.style.display = 'none';
        }
    }

    function drawButtons(matchState) {
        switch (matchState.phase) {
            case 'readyUp':
                waitOpponent.style.visibility = 'hidden';
                passButton.style.visibility = 'hidden';
                break;
            case 'main':
                if (matchState.color === matchState.turn) {
                    waitOpponent.style.visibility = 'hidden';
                    passButton.innerHTML = 'Play a card';
                    passButton.style.visibility = 'visible';
                } else {
                    waitOpponent.innerHTML = "Opponent's turn";
                    waitOpponent.style.visibility = 'visible';
                    passButton.style.visibility = 'hidden';
                }
                break;
            case 'kingPlacement':    
                passButton.style.visibility = 'hidden';
                waitOpponent.style.visibility = 'visible';
                if (matchState.public.kingPlayed) {
                    waitOpponent.innerHTML = "Opponent placing King";
                } else {
                    waitOpponent.innerHTML = "Place your King";
                }
                break;
        }
    }

    function drawWait(ctx, matchState) {
        if ((matchState.phase === 'main' && matchState.turn !== matchState.color) || 
            (matchState.phase === 'kingPlacement' && matchState.public.kingPlayed)) {
            ctx.fillStyle = 'rgba(20, 30, 100, 0.30)';
            ctx.fillRect(0, 0, board.width, board.height);    
        }
    }

    function drawWinner(ctx, winner) {
        if (winner === "none") {
            return
        }
        ctx.fillStyle = 'rgba(0, 0, 0, 0.65)';
        ctx.fillRect(0, 0, board.width, board.height);

        var messages = {"black": "Black Wins", "white": "White Wins", "draw": "Draw! Nobody Wins"};
        
        ctx.fillStyle = "white";
        ctx.font = "60px Arial";
        ctx.textAlign = "center";
        ctx.fillText(messages[winner], board.width / 2, board.height / 2 + 20);
    }

    function drawScoreboard(ctx, matchState) {
        ctx.clearRect(0, 0, scoreboard.width, scoreboard.height);

        ctx.fillStyle = "#bb3636";
        ctx.font = "11px Arial";
        ctx.textAlign = 'left';

        var white = matchState.whitePublic;
        
        var black = matchState.blackPublic;
        ctx.fillText("black turns left: ", 0, 25);
        ctx.fillText("vassal " + black.vassalTurns + ", soldier " + black.soldierTurns +  ", command " + 
            black.commandTurns, 0, 40);
        ctx.fillText("white turns left: ", 0, 60);
        ctx.fillText("vassal " + white.vassalTurns + ", soldier " + white.soldierTurns +  ", command " + 
            white.commandTurns, 0, 75);


        var x = 170;
        var y = 10;
        const textOffsetX = 230;
        textX = textOffsetX;
        textY = 33;
        var width = board.squareWidth / 3;
        var height = board.squareWidth / 3;
        const gap = 70;
        const skullOffsetX = -19;
        const skullOffsetY = -20;
    
        var pieceNames = [
            "white_Rook", "white_Knight", "white_Bishop", "white_King", 
            "black_Rook", "black_Knight", "black_Bishop", "black_King", 
        ];
        if (matchState.color === "white") {
            pieceNames = [ 
                "black_Rook", "black_Knight", "black_Bishop", "black_King", 
                "white_Rook", "white_Knight", "white_Bishop", "white_King", 
            ];    
        }
    
        // todo: get actual values from match state
        var black = matchState.blackPublic;
        var white = matchState.whitePublic;
        var hps;
        if (matchState.color === "black") {
            hps = [
                white.rook.hp, white.knight.hp, white.bishop.hp, white.king.hp, 
                black.rook.hp, black.knight.hp, black.bishop.hp, black.king.hp, 
            ];
        } else {
            hps = [
                black.rook.hp, black.knight.hp, black.bishop.hp, black.king.hp, 
                white.rook.hp, white.knight.hp, white.bishop.hp, white.king.hp, 
            ];
        }

        ctx.font = "14px Arial";
        ctx.textAlign = 'right';
        for (var i = 0; i < pieceNames.length; i++) {
            if (i === 4) {
                x = 170;
                y = 45;
                textX = textOffsetX;
                textY = y + 23;
            }
            var coords = piecesImg.pieceImageCoords[pieceNames[i]];
            ctx.drawImage(piecesImg, coords.x, coords.y, piecesImg.pieceWidth, piecesImg.pieceHeight, 
                x, y, width, height
            );  
            if (hps[i] <= 0) {
                ctx.drawImage(skullImg, 0, 0, skullImg.spriteWidth, skullImg.spriteHeight, 
                    textX + skullOffsetX, textY + skullOffsetY, board.squareWidth / 4, board.squareHeight / 4
                );
            } else {
                ctx.fillText(hps[i], textX, textY);
            }
            x += gap;
            textX += gap;    
        }
    }

    function drawPieces(ctx, matchState) {
        var flipped = matchState.color === "white";
        var pieces = matchState.board;
        var x = 0;
        var y = 0;
        const hpOffsetX = 5;
        const hpOffsetY = 15;
        const damageOffsetY = 15;
        const skullOffsetX = 82;
        const skullOffsetY = 37;
        const dmgBgWidth = 29;
        const dmgBgHeight = 17;

        var column = 0;
        for (var i = 0; i < pieces.length; i++) {
            var piece = pieces[flipped ? pieces.length - 1 - i : i];
            
            if (piece) {
                switch (piece.name) {
                    case "Jester":
                        var img = jesterBlack;
                        if (piece.color === 'white') {
                            img = jesterWhite;
                        }
                        ctx.drawImage(img, 0, 0, img.spriteWidth, img.spriteHeight,
                            x, y, board.squareWidth, board.squareHeight
                        );
                        break;
                    default:
                        var coords = piecesImg.pieceImageCoords[piece.color + "_" + piece.name];
                        ctx.drawImage(piecesImg, coords.x, coords.y, piecesImg.pieceWidth, piecesImg.pieceHeight, 
                            x, y, board.squareWidth, board.squareHeight
                        );
                }

                ctx.font = '13px Arial';
                var rightX = x + board.squareWidth - hpOffsetX;
                ctx.fillStyle = 'darkred';
                ctx.textAlign = 'right';
                ctx.fillText(piece.hp, rightX, y + hpOffsetY);
                if (piece.damage > 0) {
                    ctx.fillStyle = '#944';
                    ctx.fillRect(rightX - 24, y + 17, dmgBgWidth, dmgBgHeight);
                    ctx.fillStyle = 'white';
                    ctx.fillText(-piece.damage, rightX, y + hpOffsetY + damageOffsetY);
                }
                ctx.fillStyle = 'darkgreen';
                ctx.textAlign = 'start';
                ctx.fillText(piece.attack, x + hpOffsetX, y + hpOffsetY);

                if (piece.hp <= piece.damage) {
                    ctx.drawImage(skullImg, 0, 0, skullImg.spriteWidth, skullImg.spriteHeight, 
                        x + skullOffsetX, y + skullOffsetY, board.squareWidth / 4, board.squareHeight / 4
                    );
                }
            }

            x += board.squareWidth;
            column++;
            if (column == board.nColumns) {
                x = 0;
                column = 0;
                y += board.squareHeight;
            }
        }
    }

    function drawStatusIcons(ctx, match) {
        var flipped = match.color === "white";
        var boardStatus = match.boardStatus;
        var pieces = matchState.board;
        var x = 0;
        var y = 0;
        const squareUpOffsetX = 2;
        const squareUpOffsetY = 80;
        const squareDownOffsetX = 86;
        const squareDownOffsetY = 80;

        const squareArrowWidth = board.squareWidth / 5;
        const squareArrowHeight = board.squareHeight / 5;

        const pieceArrowWidth = board.squareWidth / 7;
        const pieceArrowHeight = board.squareHeight / 7;

        const upOffsetX = 30;
        const upOffsetY = 5;
        const downOffsetX = 60;
        const downOffsetY = 5;


        var column = 0;
        for (var i = 0; i < boardStatus.length; i++) {
            var squareStatus = boardStatus[flipped ? boardStatus.length - 1 - i : i];
            var piece = pieces[flipped ? pieces.length - 1 - i : i];
            
            if (squareStatus) {
                if (squareStatus.positive) {
                    ctx.drawImage(upArrowImg, 0, 0, upArrowImg.spriteWidth, upArrowImg.spriteHeight, 
                        x + squareUpOffsetX, y + squareUpOffsetY, squareArrowWidth, squareArrowHeight
                    );    
                }

                if (squareStatus.negative) {
                    ctx.drawImage(downArrowImg, 0, 0, downArrowImg.spriteWidth, downArrowImg.spriteHeight, 
                        x + squareDownOffsetX, y + squareDownOffsetY, squareArrowWidth, squareArrowHeight
                    );    
                }
            }
            if (piece && piece.status) {
                if (piece.status.positive) {
                    ctx.drawImage(upArrowImg, 0, 0, upArrowImg.spriteWidth, upArrowImg.spriteHeight, 
                        x + upOffsetX, y + upOffsetY, pieceArrowWidth, pieceArrowHeight
                    );    
                }

                if (piece.status.negative) {
                    ctx.drawImage(downArrowImg, 0, 0, downArrowImg.spriteWidth, downArrowImg.spriteHeight, 
                        x + downOffsetX, y + downOffsetY, pieceArrowWidth, pieceArrowHeight
                    );    
                }
            }

            x += board.squareWidth;
            column++;
            if (column == board.nColumns) {
                x = 0;
                column = 0;
                y += board.squareHeight;
            }
        }
    }

    function drawSquareHighlight(ctx, match) {
        switch (match.phase) {
            case 'kingPlacement':
            case 'main':
                var flipped = match.color === 'white';
                var highlights = match.private.highlights;
                var len = highlights.length;
                var x = 0;
                var y = 0;
                var column = 0;
                
                for (var i = 0; i < len; i++) {
                    var idx = flipped ? len - 1 - i : i;
                    switch (highlights[idx]) {
                        case highlightOff:
                            break;
                        case highlightOn:
                            ctx.fillStyle = 'rgba(255, 255, 255, 0.55)';
                            ctx.fillRect(x, y, board.squareWidth, board.squareHeight);
                            break;
                        case highlightDim:
                            ctx.fillStyle = 'rgba(0, 0, 0, 0.45)';
                            ctx.fillRect(x, y, board.squareWidth, board.squareHeight);
                            break;
                    }

                    x += board.squareWidth;
                    column++;
                    if (column == board.nColumns) {
                        x = 0;
                        column = 0;
                        y += board.squareHeight;
                    }
                }
                break;
        }
    }

    function drawBoard(ctx) {
        ctx.fillStyle = '#1ccccc';
        ctx.fillRect(0, 0, board.width, board.height);
        ctx.fillStyle = '#9fde68';
        ctx.fillRect(0, 0, board.width, board.height / 2);
    
        ctx.fillStyle = '#ef9ba9';
        var staggered = false;
        var y = 0;
        for (var i = 0; i < board.nRows; i++) {
            var x = 0; 
            if (staggered) {
                x += board.squareWidth;
            }
            for (var j = 0; j < board.nColumns / 2; j++) {
                ctx.fillRect(x, y, board.squareWidth, board.squareHeight);
                x += board.squareWidth + board.squareWidth;
            }
            y += board.squareHeight;
            staggered = !staggered;
        }
    }

    function drawCards(match) {
        var s = '';
        for (var i = 0; i < match.private.cards.length; i++) {
            var c = match.private.cards[i];
            s += '<div cardIdx="' + i + '" ';
            if (i === match.private.selectedCard) {
                s += 'class="select_card"';
            } else if (!match.private.playableCards[i]) {
                s += 'class="unplayable_card"';
            }
            if (c.type === "vassal") {
                s += '">' + cardTypes[c.type] + ' - ' + c.name + '</div>';
            } else {
                s += '">' + cardTypes[c.type] + ' - ' + c.name + ' - ' + c.rank + '</div>';
            }
            
        }
        cardList.innerHTML = s;
    }

    function drawLog(match) {
        var s = '';
        for (var i = match.log.length - 1; i >= 0; i--) {
            var entry = match.log[i];
            if (entry.startsWith("black")) {
                s += '<div class="log_entry black_log">' + entry + '</div>';
            } else if (entry.startsWith("white")) {
                s += '<div class="log_entry white_log">' + entry + '</div>';
            } else {
                s += '<div class="log_entry neutral_log">' + entry + '</div>';
            }
        }
        log.innerHTML = s;
    }
}

var lastSquare = null;
var lastPiece = null;

// square = square status, piece = piece status
function drawStatusInfo(square, piece) {
    if (square === lastSquare && piece === lastPiece) {
        return;
    }
    lastSquare = square;
    lastPiece = piece;
    var s = '';
    
    if (square.negative || square.positive) {
        s += '<h3>Square status effects:</h3>';
        let pos = square.positive;
        if (pos) {
        }
        let neg = square.negative;
        if (neg) {
            if (neg.distracted) {
                s += '<div class="status_entry negative">Distracted: piece in this square will not attack</div>';
            }
        }
    }

    if (piece) {
        s += '<h3>Piece status effects:</h3>';
        let pos = piece.positive;
        if (pos) {
            if (pos.amplify > 0) {
                s += '<div class="status_entry positive">Amplify: piece inflicts double damage. Remaining rounds: ' + pos.amplify + '</div>';
            }
            if (pos.armor > 0) {
                s += '<div class="status_entry positive">Armor: reduces incoming damage. Strength: ' + pos.armor + '</div>';
            }
            if (pos.damageImmune > 0) {
                s += '<div class="status_entry positive">Damage Immune: piece cannot take damage. Remaining rounds: ' + pos.damageImmune + '</div>';
            }
        }
        let neg = piece.negative;
        if (neg) {
            if (neg.vulnerability > 0) {
                s += '<div class="status_entry negative">Vulnerability: piece takes double damage. Remaining rounds: ' + neg.vulnerability + '</div>';
            }
            if (neg.poison > 0) {
                s += '<div class="status_entry negative">Poison: piece takes damage every round in combat. HP removed: ' + neg.poison + '</div>';
            }
            if (neg.unreclaimable > 0) { 
                s += '<div class="status_entry negative">Unreclaimable: piece cannot be reclaimed. Remaining rounds: ' + neg.unreclaimable + '</div>';
            }
            if (neg.distracted > 0) {
                s += '<div class="status_entry negative">Distracted: piece will not inflict damage. Remaining rounds: ' + neg.distracted + '</div>';
            }
            if (neg.enraged > 0) {
                s += '<div class="status_entry negative">Enraged: piece attacks allies as well as enemies. Remaining rounds: ' + neg.enraged + '</div>';
            }
            if (neg.transparent > 0) {
                s += '<div class="status_entry negative">Transparency: piece does not block attacks. Remaining rounds: ' + neg.transparent + '</div>';
            }
        }
    }
    
    statusInfo.innerHTML = s;
}


function drawTimer(match) {
    switch (matchState.phase) {
        case 'main':
        case 'kingPlacement':
            var seconds = Math.floor(match.turnRemainingMilliseconds / 1000);
            timer.innerHTML = ((seconds < 0) ? 0 : seconds) + ' seconds';
            break;
    }
}



// *** logic ***

var timerHandle;
var timeSincePing;
const interval = 1000;
const pingInterval = 20000;

function setTimers(match) {
    window.clearInterval(timerHandle);
    timeSincePing = 0;
    
    switch (match.phase) {
        case 'kingPlacement':
        case 'main':    
            timerHandle = window.setInterval(
                function () {
                    match.turnRemainingMilliseconds -= interval;
                    drawTimer(match);

                    // extra half second as cushion (don't want to 
                    // send too early or else server ignores event)
                    if (match.turnRemainingMilliseconds + 500 < 0) {
                        conn.send("time_expired ");
                        waitingResponse = true;
                    }

                    timeSincePing += interval;
                    if (timeSincePing > pingInterval) {
                        conn.send("ping ");
                        timeSincePing = 0;
                    }
                },
                interval
            );
            break;
    }
}

readyupButton.addEventListener('click', function (evt) {
    switch (matchState.phase) {
        case 'readyUp':
            conn.send("ready " );
            waitingResponse = true;
            break;
    }
}, false);

cardList.addEventListener('mousedown', function (evt) {
    switch (matchState.phase) {
        case 'main':
            if (waitingResponse || (matchState.color !== matchState.turn)) {
                return; // not your turn!
            }
            var idx = evt.target.getAttribute('cardIdx');
            if (idx === '' || idx === null) {
                return;
            }
            conn.send("click_card " + JSON.stringify({selectedCard: parseInt(idx)}));
            waitingResponse = true;
            break;
    }
}, false);

cardList.addEventListener('mouseleave', function (evt) {
    switch (matchState.phase) {
        case 'main':
        case 'kingPlacement':
            for (var c of cardList.children) {
                c.classList.remove('highlight_card');
            }
            cardDescription.style.display = 'none';
            statusInfo.style.display = 'none';
            logBox.style.display = 'block';
            break;
    }
}, false);

cardList.addEventListener('mouseover', function (evt) {
    switch (matchState.phase) {
        case 'main':
        case 'kingPlacement':
            var idx = evt.target.getAttribute('cardIdx');
            if (idx === '' || idx === null) {
                cardDescription.style.display = 'none';
                statusInfo.style.display = 'none';
                logBox.style.display = 'block';
                return;
            }
            cardDescription.innerHTML = cardDescriptions[matchState.private.cards[idx].name];
            cardDescription.style.display = 'block';
            logBox.style.display = 'none';
            statusInfo.style.display = 'none';
            for (var c of cardList.children) {
                if (c === evt.target) {
                    c.classList.add('highlight_card');
                } else {
                    c.classList.remove('highlight_card');
                }
            }
            break;
    }
}, false);


function updateSquareInfoBox(clientX, clientY) {
    switch (matchState.phase) {
        case 'main':
        case 'kingPlacement':
            if (clientX === null) {
                return;
            }
            var rect = canvas.getBoundingClientRect();
            var mouseX = clientX - rect.left;
            var mouseY = clientY - rect.top;
        
            var squareX = Math.floor(mouseX / board.squareWidth);
            var squareY = Math.floor(mouseY / board.squareHeight);

            if (squareX < 0) {
                squareX = 0;
            } else if (squareX >= board.nColumns) {
                squareX = board.nColumns - 1;
            }
            if (squareY < 0) {
                squareY = 0;
            } else if (squareY >= board.nRows) {
                squareY = board.nRows - 1;
            }
        
            // invert for white player
            if (matchState.color === "white") {
                squareX = board.nColumns - 1 - squareX;
                squareY = board.nRows - 1 - squareY;
            }

            var idx = squareX + squareY * board.nColumns;
            var squareStatus = matchState.boardStatus[idx];
            var piece = matchState.board[idx];
            var pieceStatus = null;
            if (piece) {
                pieceStatus = piece.status;
            }
            if (squareStatus.positive || squareStatus.negative || pieceStatus) {
                drawStatusInfo(squareStatus, pieceStatus);
                cardDescription.style.display = 'none';
                logBox.style.display = 'none';
                statusInfo.style.display = 'block';
            } else {
                cardDescription.style.display = 'none';
                logBox.style.display = 'block';
                statusInfo.style.display = 'none';
            }
            break;
    }
}

var boardClientX = null;
var boardClientY = null;

canvas.addEventListener('mousemove', function (evt) {
    boardClientX = evt.clientX;
    boardClientY = evt.clientY;
    updateSquareInfoBox(boardClientX, boardClientY);
    
}, false);

canvas.addEventListener('mouseleave', function (evt) {
    boardClientX = null;
    boardClientY = null;
    cardDescription.style.display = 'none';
    logBox.style.display = 'block';
    statusInfo.style.display = 'none';
}, false);

canvas.addEventListener('mousedown', function (evt) {
    if (waitingResponse) {
        return; // not your turn!
    }
    switch (matchState.phase) {
        case 'main':
            if (matchState.color !== matchState.turn) {
                return; // not your turn!
            }
        case 'kingPlacement':
            var rect = canvas.getBoundingClientRect();
            var mouseX = evt.clientX - rect.left;
            var mouseY = evt.clientY - rect.top;
        
            var squareX = Math.floor(mouseX / board.squareWidth);
            var squareY = Math.floor(mouseY / board.squareHeight);

            if (squareX < 0) {
                squareX = 0;
            } else if (squareX >= board.nColumns) {
                squareX = board.nColumns - 1;
            }
            if (squareY < 0) {
                squareY = 0;
            } else if (squareY >= board.nRows) {
                squareY = board.nRows - 1;
            }
        
            // invert for white player
            if (matchState.color === "white") {
                squareX = board.nColumns - 1 - squareX;
                squareY = board.nRows - 1 - squareY;
            }
     
            conn.send("click_board " + JSON.stringify({x: squareX, y: squareY}));    
            waitingResponse = true;
            break;
    }
}, false);