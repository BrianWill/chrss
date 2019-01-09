var canvas = document.getElementById('board');
var ctx = canvas.getContext('2d');

const board = {
    width: 800,
    height: 800,
    nRows: 8,
    nColumns: 8,
};
board.squareHeight = board.height / board.nRows;
board.squareWidth = board.width / board.nColumns;
Object.freeze(board);


var pieces = new Image();
pieces.pieceHeight = 45;
pieces.pieceWidth = 45;
pieces.src = "/static/pieces.svg";
pieces.pieceImageCoords = {
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

pieces.onload = function () {
    drawBoard(ctx);
    drawPiece(ctx, 'white_queen', 5, 3);
    drawPiece(ctx, 'black_king', 0, 0);
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


function drawPiece(ctx, piece, x, y) {
    var coords = pieces.pieceImageCoords[piece];
    ctx.drawImage(pieces, coords.x, coords.y, pieces.pieceWidth, pieces.pieceHeight, 
        x * board.squareWidth, y * board.squareHeight, board.squareWidth, board.squareHeight
    );
}


var url = 'ws://localhost:5000/ws';
var c = new WebSocket(url);

var send = function(data){
  console.log(new Date() + " ==> "+data+"\n");
  c.send(data)
}

c.onmessage = function(msg){
  console.log(new Date() + " <== "+msg.data+"\n");
  console.log(msg)
}

c.onopen = function(){
  setInterval( 
    function(){ send("ping") }
  , 1000 )
}
