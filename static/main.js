var canvas = document.getElementById('board');
var ctx = canvas.getContext('2d');

const board = {
    width: 600,
    height: 600,
    nRows: 8,
    nColumns: 8,
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
}

piecesImg.onload = function () {
    drawBoard(ctx);
    //drawPiece(ctx, 'white_queen', 5, 3);
    //drawPiece(ctx, 'black_king', 0, 0);
}

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


var matchId = window.location.pathname.substring(7);
var url = 'ws://localhost:5000/ws/' + matchId;
var c = new WebSocket(url);

var send = function(data){
    console.log(new Date() + " ==> "+data+"\n");
    c.send(data);
}

function drawPieces(ctx, pieces) {
    for (var i = 0; i < pieces.length; i++) {
        var piece = pieces[i];
        var coords = piecesImg.pieceImageCoords[piece.color + "_" + piece.type];
        ctx.drawImage(piecesImg, coords.x, coords.y, piecesImg.pieceWidth, piecesImg.pieceHeight, 
            piece.x * board.squareWidth, piece.y * board.squareHeight, board.squareWidth, board.squareHeight
        );
    }
}

c.onmessage = function(msg){
    console.log(" <== " + new Date() + " <== \n");
    console.log(msg);
    if (msg.data === "Match is full.") {
        alert("Cannot join match. Match already has two players.");
        return;
    }
    var match = JSON.parse(msg.data);
    drawBoard(ctx);
    drawPieces(ctx, match.pieces);
}

c.onerror = function(err) {
    console.log(new Date() + " error: "+err.data+"\n");
    console.log(err);
  }

c.onopen = function(){
    setInterval(
        function(){ send("ping"); },
        2000
    );
}
