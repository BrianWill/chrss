var canvas = document.getElementById('board');
var ctx = canvas.getContext('2d');
var scoreboard = document.getElementById('scoreboard');
var scoreboardCtx = scoreboard.getContext('2d');

const fanfare = document.getElementById('fanfare');
fanfare.volume = 0.5;

var cardList = document.getElementById('card_list');
var cardDescription = document.getElementById('card_description');
var passButton = document.getElementById('pass_button');
var waitOpponent = document.getElementById('wait_opponent');
var timer = document.getElementById('timer');
var log = document.getElementById('log');

var matchState;

const NO_SELECTED_CARD = -1;
const board = {
    width: 600,
    height: 600,
    nRows: 6,
    nColumns: 6,
};
board.squareHeight = board.height / board.nRows;
board.squareWidth = board.width / board.nColumns;
Object.freeze(board);


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

var cardDescriptions = {
    'King': `<h3>King: 0 mana cost, 40 starting hp</h3>
<div>Game is won by destroying enemy king. Can attack once per round. Attacks one space in all directions.</div>`,
    'Rook': `<h3>Rook: 0 mana cost, 10 starting hp</h3>
<div>Can attack once per round. Attacks in the four cardinal directions (up, down, left, right).</div>`,
    'Bishop': `<h3>Bishop: 0 mana cost, 10 starting hp</h3>
<div>Can attack once per round. Attacks diagonally.</div>`,
    'Knight': `<h3>Knight: 0 mana cost, 10 starting hp</h3>
<div>Can attack once per round. Attacks are not blocked by other units. Attacks in 'L' shape: two spaces in cardinal direction and one space over.</div>`,
    'Pawn': `<h3>Pawn: 0 mana cost, 3 starting hp</h3>
<div>Can attack once per round. Attacks one space diagonally towards opponent side.</div>`,
};



var matchId = window.location.pathname.substring(7);
var url = 'ws://localhost:5000/ws/' + matchId;
var conn = new WebSocket(url);


var waitingResponse = false;


conn.onmessage = function(msg){
    console.log(" <== " + new Date() + " <== \n");
    console.log(msg);
    if (msg.data === "Match is full.") {
        cardList.innerHTML = "Cannot join match. Match already has two players.";
        return;
    }
    matchState = JSON.parse(msg.data);
    if (matchState.color === 'black') {
        matchState.public = matchState.blackPublic;
    } else {
        matchState.public = matchState.whitePublic;
    }
    setTimers(matchState);
    draw(matchState);
    if (matchState.newRound) {
        fanfare.play();
    }
    waitingResponse = false;
}

conn.onerror = function(err) {
    console.log(new Date() + " error: "+err.data+"\n");
    console.log(err);
  }

conn.onopen = function(){
    conn.send("get_state ");
}


//  *** draw ***

function draw(matchState) {
    drawBoard(ctx);
    drawPieces(ctx, matchState);
    drawSquareHighlight(ctx, matchState);
    drawWait(ctx, matchState);
    drawWinner(ctx, matchState.winner);
    drawCards(matchState);
    drawScoreboard(scoreboardCtx, matchState);
    drawButtons(matchState);
    drawTimer(matchState);

    function drawButtons(matchState) {
        if (matchState.color === matchState.turn && matchState.public.kingPlayed) {
            passButton.style.display = 'block';
            if (matchState.passPrior) {
                passButton.innerHTML = 'End round';
            } else {
                passButton.innerHTML = 'Pass';
            }
        } else {
            passButton.style.display = 'none';
        }
        if (matchState.color === matchState.turn) {
            waitOpponent.style.display = 'none';
        } else {
            waitOpponent.style.display = 'block';
        }
    }

    function drawWait(ctx, matchState) {
        if (matchState.winner === 'none' && matchState.turn !== matchState.color) {
            ctx.fillStyle = 'rgba(0, 30, 100, 0.20)';
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
        ctx.font = "18px Arial";
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
        const maxX = board.nColumns * board.squareWidth;
        var x = 0;
        var y = 0;
        const hpOffsetX = 5;
        const hpOffsetY = 15;
        const damageOffsetY = 15;
        const skullOffsetX = 74;
        const skullOffsetY = 37;

        for (var i = 0; i < pieces.length; i++) {
            var piece = pieces[flipped ? pieces.length - 1 - i : i];

            if (piece) {
                var coords = piecesImg.pieceImageCoords[piece.color + "_" + piece.name];
                ctx.drawImage(piecesImg, coords.x, coords.y, piecesImg.pieceWidth, piecesImg.pieceHeight, 
                    x, y, board.squareWidth, board.squareHeight
                );

                ctx.font = '13px Arial';
                var rightX = x + board.squareWidth - hpOffsetX;
                ctx.fillStyle = 'darkred';
                ctx.textAlign = 'right';
                ctx.fillText(piece.hp, rightX, y + hpOffsetY);
                if (piece.damage > 0) {
                    ctx.fillStyle = 'red';
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
            if (x == maxX) {
                x = 0;
                y += board.squareHeight;
            }
        }
    }

    function drawSquareHighlight(ctx, matchState) {
        if (!matchState.private.highlightEmpty) {
            return
        }
        var flipped = matchState.color === 'white';
        var pieces = matchState.board;
        const maxX = board.nColumns * board.squareWidth;
        var x = 0;
        var y = 0;
        ctx.fillStyle = 'rgba(0, 0, 0, 0.45)';
        for (var i = 0; i < pieces.length; i++) {
            var idx = flipped ? pieces.length -1 - i : i;
            if (pieces[idx] !== null || i < pieces.length / 2) {
                ctx.fillRect(x, y, board.squareWidth, board.squareHeight);
            }
            x += board.squareWidth;
            if (x == maxX) {
                x = 0;
                y += board.squareHeight;
            }
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
            }
            s += '">' + c.manaCost + ' - ' + c.name + '</div>';
        }
        cardList.innerHTML = s;
    }
}

function drawTimer(match) {
    timer.innerHTML = Math.floor(match.turnRemainingMilliseconds / 1000) + ' seconds';
}



// *** logic ***

var timerTimeoutHandle;
var timerIntervalHandle;

function setTimers(match) {
    window.clearTimeout(timerTimeoutHandle);
    window.clearInterval(timerIntervalHandle);
    if (match.turnRemainingMilliseconds < 0) {
        conn.send("time_expired ");
        waitingResponse = true;
        return
    }
    timerTimeoutHandle = window.setTimeout(
        function () {
            conn.send("time_expired ");
            waitingResponse = true;
        },
        match.turnRemainingMilliseconds
    );
    timerIntervalHandle = window.setInterval(
        function () {
            match.turnRemainingMilliseconds -= 1000;
            drawTimer(match);
        }, 
        1000
    );
}




cardList.addEventListener('mousedown', function (evt) {
    if (matchState.winner !== 'none') {
        return
    }
    if (waitingResponse || (matchState.color !== matchState.turn)) {
        return; // not your turn!
    }
    var idx = evt.target.getAttribute('cardIdx');
    if (idx === '' || idx === null) {
        return;
    }
    conn.send("click_card " + JSON.stringify({selectedCard: parseInt(idx)}));
    waitingResponse = true;
}, false);

cardList.addEventListener('mouseleave', function (evt) {
    if (matchState.winner !== 'none') {
        return
    }
    for (var c of cardList.children) {
        c.classList.remove('highlight_card');
    }
    var private = matchState.private;
    if (private.selectedCard !== NO_SELECTED_CARD) {
        cardDescription.innerHTML = cardDescriptions[private.cards[private.selectedCard].name];
    } else {
        cardDescription.innerHTML = '';
    }
}, false);

cardList.addEventListener('mouseover', function (evt) {
    if (matchState.winner !== 'none') {
        return
    }
    var idx = evt.target.getAttribute('cardIdx');
    if (idx === '' || idx === null) {
        return;
    }
    cardDescription.innerHTML = cardDescriptions[matchState.private.cards[idx].name];
    for (var c of cardList.children) {
        if (c === evt.target) {
            c.classList.add('highlight_card');
        } else {
            c.classList.remove('highlight_card');
        }
    }
}, false);


canvas.addEventListener('mousedown', function (evt) {
    if (matchState.winner !== 'none') {
        return
    }
    if (waitingResponse || (matchState.color !== matchState.turn)) {
        return; // not your turn!
    }

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
}, false);

passButton.addEventListener('click', function (evt) {
    if (matchState.winner !== 'none') {
        return
    }
    console.log("passed");
    conn.send("pass ");
    waitingResponse = true;
}, false);