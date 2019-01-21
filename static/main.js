var canvas = document.getElementById('board');
var ctx = canvas.getContext('2d');
var scoreboard = document.getElementById('scoreboard');
var scoreboardCtx = scoreboard.getContext('2d');
var turn = document.getElementById('turn');
var cardList = document.getElementById('card_list');
var cardDescription = document.getElementById('card_description');
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
    'white_king': {x: 0, y: 0},
    'white_queen': {x: 45, y: 0},
    'white_bishop' : {x: 90, y: 0},
    'white_knight' : {x: 135, y: 0},
    'white_rook' : {x: 180, y: 0},
    'white_pawn' : {x: 225, y: 0},
    'black_king': {x: 0, y: 45},
    'black_queen': {x: 45, y: 45},
    'black_bishop' : {x: 90, y: 45},
    'black_knight' : {x: 135, y: 45},
    'black_rook' : {x: 180, y: 45},
    'black_pawn' : {x: 225, y: 45},
};

piecesImg.onload = function (evt) {
    if (matchState) {
        render(matchState);
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
        turn.innerHTML = "Cannot join match. Match already has two players.";
        return;
    }
    matchState = JSON.parse(msg.data);
    render(matchState);
    waitingResponse = false;
}

conn.onerror = function(err) {
    console.log(new Date() + " error: "+err.data+"\n");
    console.log(err);
  }

conn.onopen = function(){
    conn.send("get_state " + JSON.stringify({date: new Date()}));
}


//  *** render ***

function drawBoard(ctx) {
    ctx.fillStyle = '#33FFFF';
    ctx.fillRect(0, 0, board.width, board.height);
    ctx.fillStyle = 'pink';
    
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

function displayTurn(match) {
    if (match.color === match.turn) {
        turn.innerHTML = "Your turn (" + match.color + "). <br/> Select a card to play or select one of your pieces to have it attack an enemy.";
    } else {
        turn.innerHTML = "Opponent's turn (" + match.color + ").";
    }
    
}

function displayCards(match) {
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

function render(matchState) {
    drawBoard(ctx);
    drawPieces(ctx, matchState.pieces);
    displayTurn(matchState);
    displayCards(matchState);
    drawScoreboard(scoreboardCtx);
}

function drawScoreboard(ctx, matchState) {
    var x = 20;
    var y = 10;
    var textX = 56;
    var textY = 35;
    var width = board.squareWidth / 3;
    var height = board.squareWidth / 3;
    const gap = 70;

    var pieceNames = ["white_king", "white_rook", "white_knight", "white_bishop", 
        "black_king", "black_rook", "black_knight", "black_bishop", 
    ];

    // todo: get actual values from match state
    var hps = [40, 10, 10, 10, 40, 10, 10, 10];

    for (var i = 0; i < pieceNames.length; i++) {
        if (i === 4) {
            x = 20;
            y = 45;
            textX = 56;
            textY = y + 25;
        }
        var coords = piecesImg.pieceImageCoords[pieceNames[i]];
        ctx.drawImage(piecesImg, coords.x, coords.y, piecesImg.pieceWidth, piecesImg.pieceHeight, 
            x, y, width, height
        );  
        ctx.fillStyle = "#bb3636";
        ctx.font = "18px Arial";
        ctx.fillText(hps[i], textX, textY);
        x += gap;
        textX += gap;    
    }
}



function highlightSquares(ctx, matchState) {
    // if no selected square, drawing {x: -1, x: -1} is clipped
    ctx.fillStyle = 'rgba(0, 0, 0, 0.4)';
    var x = matchState.private.selectedPos.x * board.squareWidth;
    var y = matchState.private.selectedPos.y * board.squareHeight;
    ctx.fillRect(x, y, board.squareWidth, board.squareHeight);

    // highlight valid attack targets
    ctx.fillStyle = 'rgba(255, 255, 120, 0.5)';
    var moves = matchState.private.attackPos;
    for (var m of moves) {
        var x = m.x * board.squareWidth;
        var y = m.y * board.squareHeight;
        ctx.fillRect(x, y, board.squareWidth, board.squareHeight);
    }
}

function drawPieces(ctx, pieces) {
    for (var i = 0; i < pieces.length; i++) {
        var piece = pieces[i];
        var coords = piecesImg.pieceImageCoords[piece.color + "_" + piece.type];
        ctx.drawImage(piecesImg, coords.x, coords.y, piecesImg.pieceWidth, piecesImg.pieceHeight, 
            piece.pos.x * board.squareWidth, piece.pos.y * board.squareHeight, board.squareWidth, board.squareHeight
        );
    }
}



// *** logic ***


cardList.addEventListener('mousedown', function (evt) {
    if (waitingResponse || (matchState.color !== matchState.turn)) {
        return; // not your turn!
    }
    var idx = evt.target.getAttribute('cardIdx');
    if (idx === '' || idx === null) {
        return;
    }
    conn.send("click_card " + JSON.stringify({date: new Date(), selectedCard: parseInt(idx)}));
    waitingResponse = true;
}, false);

cardList.addEventListener('mouseleave', function (evt) {
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
    if (waitingResponse || (matchState.color !== matchState.turn)) {
        return; // not your turn!
    }

    var rect = canvas.getBoundingClientRect();
    var mouseX = evt.clientX - rect.left;
    var mouseY = evt.clientY - rect.top;
    console.log("canvas click: " + mouseX + ", " + mouseY);

    var squareX = Math.floor(mouseX / board.squareWidth);
    var squareY = Math.floor(mouseY / board.squareHeight);
    
    conn.send("click_board " + JSON.stringify({date: new Date(), x: squareX, y: squareY}));
    waitingResponse = true;
}, false);