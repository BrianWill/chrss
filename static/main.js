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
const reclaimPhase = 'reclaim';
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


var cardDescriptions = {
    'Rook': `<h3>Rook: 0 mana cost, 20 HP, 6 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks up/down/left/right. You only get one Rook in the match. When reclaimed, its HP and status effects persist, and you get a Rook card back in your hand. When reclaimed, healed for 5 HP.</div>`,
    'Bishop': `<h3>Bishop: 0 mana cost, 25 HP, 4 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks diagonally. You only get one Bishop in the match. When reclaimed, its HP and status effects persist, and you get a Bishop card back in your hand.</div>`,
    'Knight': `<h3>Knight: 0 mana cost, 25 HP, 5 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks are not blocked by other units. Attacks in 'L' shape: two spaces in cardinal direction and one space over. You only get one Knight in the match. When reclaimed, its HP and status effects persist, and you get a Knight card back in your hand.</div>`,
    'Pawn': `<h3>Pawn: 0 mana cost, 5 HP, 2 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks one space diagonally towards opponent side.</div>`,
    'Queen': `<h3>Queen: 5 mana cost, 15 HP, 6 attack</h3>
<div>Click free square on your side to place.</div><br/><div>Attacks diagonally and up/down/left/right.</div>`,
    'Castle': `<h3>Castle: 2 mana cost</h3>
<div>Click either King.<br/><br/>Swaps the clicked King's position with the Rook of the same color. (Can only use Castle on a King whose Rook is on the board.)</div>`,
    'Reclaim Vassal': `<h3>Reclaim Vassal: 2 mana cost</h3>
<div>Click a Knight, Bishop, or Rook.<br/><br/>The clicked vassal is reclaimed immediately.</div>`,
    'Swap Front Lines': `<h3>Swap Front Lines: 2 mana cost</h3>
<div>Click a King.<br/><br/>Swaps all pieces between the front and middle rows on the clicked King's side.</div>`,
    'Remove Pawn': `<h3>Remove Pawn: 2 mana cost</h3>
<div>Click a Pawn of either color to remove.</div>`,
    'Force Combat': `<h3>Force Combat: 3 mana cost</h3>
<div>Click your King to confirm.<br/><br/>Immediately advances match to combat and the end of round.</div>`,
    'Mirror': `<h3>Mirror: 2 mana cost</h3>
<div>Click either King.<br/><br/>Moves all pieces of clicked color to their horizontally-mirrored positions, <em>e.g.</em> all pieces in the leftmost column move to the rightmost column and <em>vice versa</em>.</div>`,
    'Heal': `<h3>Heal: 2 mana cost</h3>
<div>Click any of your pieces (except your King).<br/><br/>Adds 5 HP to a non-King piece (not capped by the piece's starting health).</div>`,
    'Toggle Pawn': `<h3>Toggle Pawn: 2 mana cost</h3>
<div>Click a Pawn.<br/><br/>Moves a Pawn in the front row to the middle row or moves a Pawn in the middle row to the front row. The destination square must be unoccupied.</div>`,
    'Drain Mana': `<h3>Drain Mana: 2 mana cost</h3>
<div>Click enemy King.<br/><br/>Subtract two mana from the enemy's current mana.</div>`,
    'Nuke': `<h3>Nuke: 2 mana cost</h3>
<div>Click a King.<br/><br/>Immediately inflict 6 damage on all pieces within 1 square of the clicked King and 3 damage on all pieces withing 2 squares of the clicked King.</div>`,
    'Shove': `<h3>Shove: 2 mana cost</h3>
<div>Click a piece.<br/><br/>Moves a white piece one square towards white's back row; moves a black piece one square towards black's back row.</div>`,
    'Advance': `<h3>Advance: 2 mana cost</h3>
<div>Click a piece.<br/><br/>Moves a white piece one square towards black's back row; moves a black piece one square towards white's back row.</div>`,
    'Restore Mana': `<h3>Restore Mana: 2 mana cost</h3>
<div>Click your King.<br/><br/>Restores your mana to max.</div>`,
    'Summon Pawn': `<h3>Summon Pawn: 2 mana cost</h3>
<div>Click your King.<br/><br/>Summons an additional pawn (subject to usual max of 5 pawns and restrictions on pawn placement).</div>`,
    'Jester': `<h3>Jester: 3 mana cost, 12 HP, 0 attack</h3>
<div>Click free square on your side to place.<br/><br/>Does not attack. Puts 'distract' effect on all adjacent squares except those behind the jester. A piece in a square with distract does not attack.</div>`,
    'Vulnerability': `<h3>Vulnerability: 2 mana cost</h3>
<div>Click enemy piece.<br/><br/>Doubles damage the targeted piece takes this round.</div>`,
    'Amplify': `<h3>Amplify: 2 mana cost</h3>
<div>Click allied piece.<br/><br/>Doubles damage the targeted piece inflicts this round.</div>`,
};



var matchId = window.location.pathname.substring(7);
var url = 'ws://chrss-game.herokuapp.com/ws/' + matchId;
if (location.hostname == 'localhost') {
    url = 'ws://localhost:5000/ws/' + matchId;
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
    if (!matchState.log) {
        matchState.log = [];
    }
    setTimers(matchState);
    draw(matchState);

    // sounds
    try {
        if (matchState.newRound) {
            fanfare.play();
        } else if (matchState.newTurn) {
            switch (matchState.phase) {
                case 'reclaim':
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
    console.log("Connection close " + new Date() + err);
    console.log(err);
}

conn.onopen = function(){
    conn.send("get_state ");
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
                    passButton.innerHTML = 'Pass';
                    if (matchState.passPrior) {
                        passButton.innerHTML = 'Combat!';
                    }
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
            case 'reclaim':
                waitOpponent.style.visibility = 'visible';
                if (matchState.public.reclaimSelectionMade) {
                    waitOpponent.innerHTML = "Waiting for opponent";
                    passButton.style.visibility = 'hidden';
                } else {
                    waitOpponent.innerHTML = "Select 0 to 2 pieces";
                    passButton.innerHTML = 'Reclaim selected piece(s)';
                    passButton.style.visibility = 'visible';
                }
                break;
        }
    }

    function drawWait(ctx, matchState) {
        if ((matchState.phase === 'main' && matchState.turn !== matchState.color) || 
            (matchState.phase === 'reclaim' && matchState.public.reclaimSelectionMade) ||
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

        // draw mana
        ctx.fillStyle = "#5277a9";
        ctx.font = "16px Arial";
        ctx.textAlign = 'start';
        if (matchState.color === "white") {
            var textX = 10;
            var textY = 35;
            ctx.fillText('black mana  ' + matchState.blackPublic.manaCurrent + ' / ' + matchState.blackPublic.manaMax, textX, textY);
            textX = 10;
            textY = 70;
            ctx.fillText('white mana  ' + matchState.whitePublic.manaCurrent + ' / ' + matchState.whitePublic.manaMax, textX, textY);
        } else {
            var textX = 10;
            var textY = 35;
            ctx.fillText('white mana  ' + matchState.whitePublic.manaCurrent + ' / ' + matchState.whitePublic.manaMax, textX, textY);
            textX = 10;
            textY = 70;
            ctx.fillText('black mana  ' + matchState.blackPublic.manaCurrent + ' / ' + matchState.blackPublic.manaMax, textX, textY);
        }

        //
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
                white.rookHP, white.knightHP, white.bishopHP, white.kingHP, 
                black.rookHP, black.knightHP, black.bishopHP, black.kingHP, 
            ];
        } else {
            hps = [
                black.rookHP, black.knightHP, black.bishopHP, black.kingHP, 
                white.rookHP, white.knightHP, white.bishopHP, white.kingHP, 
            ];
        }

        ctx.fillStyle = "#bb3636";
        ctx.font = "18px Arial";
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
            case 'reclaim':
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
            s += '">' + c.manaCost + ' - ' + c.name + '</div>';
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
                let round = (pos.amplify === 1) ? 'round' : 'rounds';
                s += '<div class="status_entry positive">Amplify: piece inflicts double damage for ' +
                pos.amplify + ' ' + round + '</div>';
            }
        }
        let neg = piece.negative;
        if (neg) {
            if (neg.vulnerability > 0) {
                let round = (neg.vulnerability === 1) ? 'round' : 'rounds';
                s += '<div class="status_entry negative">Vulnerability: piece takes double damage for ' +
                     neg.vulnerability + ' ' + round + '</div>';
            }
        }
    }
    
    statusInfo.innerHTML = s;
}


function drawTimer(match) {
    switch (matchState.phase) {
        case 'main':
        case 'reclaim':
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
        case 'reclaim':
        case 'kingPlacement':
        case 'main':    
            timerHandle = window.setInterval(
                function () {
                    match.turnRemainingMilliseconds -= interval;
                    drawTimer(match);

                    // extra half second as cushion (don't want to 
                    // send too early or else server ignores event)
                    if (match.turnRemainingMilliseconds + 500 < 0) {
                        conn.send(match.phase === 'reclaim' ? "reclaim_time_expired " : "time_expired ");
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
        case 'reclaim':
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
        case 'reclaim':
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


canvas.addEventListener('mousemove', function (evt) {
    switch (matchState.phase) {
        case 'main':
        case 'reclaim':
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
}, false);

canvas.addEventListener('mouseleave', function (evt) {
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
        case 'reclaim':
        case 'kingPlacement':
            var rect = canvas.getBoundingClientRect();
            var mouseX = evt.clientX - rect.left;
            var mouseY = evt.clientY - rect.top;
        
            var squareX = Math.floor(mouseX / board.squareWidth);
            var squareY = Math.floor(mouseY / board.squareHeight);
        
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

passButton.addEventListener('click', function (evt) {
    switch (matchState.phase) {
        case 'main':
            conn.send("pass ");
            waitingResponse = true;
            break;
        case 'reclaim':
            conn.send("reclaim_done ")
    }
}, false);