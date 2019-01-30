package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	white = "white"
	black = "black"
	draw  = "draw"
	none  = "none"
)

const (
	pawn   = "Pawn"
	king   = "King"
	queen  = "Queen"
	rook   = "Rook"
	bishop = "Bishop"
	knight = "Knight"
)

const (
	pawnHP       = 5
	pawnAttack   = 2
	kingHP       = 50
	kingAttack   = 12
	bishopHP     = 25
	bishopAttack = 4
	bishopMana   = 0
	knightHP     = 25
	knightAttack = 5
	knightMana   = 0
	rookHP       = 20
	rookAttack   = 6
	rookMana     = 0
	queenHP      = 15
	queenAttack  = 6
	queenMana    = 3
)

const (
	castleCard         = "Castle"
	castleMana         = 2
	reclaimVassalCard  = "Reclaim Vassal"
	reclaimVassalMana  = 2
	swapFrontLinesCard = "Swap Front Lines"
	swapFrontLinesMana = 2
	removePawnCard     = "Remove Pawn"
	removePawnMana     = 2
	forceCombatCard    = "Force Combat"
	forceCombatMana    = 3
)

const reclaimHealRook = 5

const defaultInstruction = "Pick a card to play or pass."
const kingInstruction = "Pick a square to place your king."

const nColumns = 6
const nRows = 6

const turnTimer = 50 * int64(time.Second)
const maxConcurrentMatches = 100

const (
	highlightOff = iota
	highlightOn
	highlightDim
)

type Phase string

const (
	readyUpPhase       Phase = "readyUp"
	mainPhase          Phase = "main"
	kingPlacementPhase Phase = "kingPlacement"
	reclaimPhase       Phase = "reclaim"
	gameoverPhase      Phase = "gameover"
)

const maxReclaim = 2 // max number of pieces to reclaim at end of round
const matchTimeout = 20 * int64(time.Minute)

type Match struct {
	Name      string // used to identify the match in browser
	BlackConn *websocket.Conn
	WhiteConn *websocket.Conn
	Mutex     sync.RWMutex
	// rows stored in order top-to-bottom, e.g. nColumns is index of leftmost square in second row
	// (*Pierce better for empty square when JSONifying; Board[i] points to pieces[i]; the array is here simply for memory locality)
	pieces         [nColumns * nRows]Piece  // zero value for empty square
	Board          [nColumns * nRows]*Piece // nil for empty square
	CommunalCards  []Card                   // card in pool shared by both players
	UUID           string
	BlackPrivate   PrivateState
	WhitePrivate   PrivateState
	BlackPublic    PublicState
	WhitePublic    PublicState
	Turn           string // white, black
	PassPrior      bool   // true if prior move was a pass
	FirstTurnColor string // color of player who had first turn this round
	Round          int    // starts at 1
	Winner         string // white, black, none, draw
	StartTime      int64  // unix time
	LastMoveTime   int64  // should be initialized to match start time
	Log            []string
	Phase          Phase
}

// info a player doesn't want opponent to see
type PrivateState struct {
	Cards             []Card                `json:"cards"`
	SelectedCard      int                   `json:"selectedCard"`  // index into cards slice
	PlayableCards     []bool                `json:"playableCards"` // parallel to Cards
	Highlights        [nColumns * nRows]int `json:"highlights"`
	PlayerInstruction string                `json:"playerInstruction"`
	ReclaimSelections []Pos                 `json:"reclaimSelections"`
	Other             *PrivateState         `json:"-"`
}

type PublicState struct {
	Ready                bool         `json:"ready"` // match does not start until both player's are ready
	ReclaimSelectionMade bool         `json:"reclaimSelectionMade"`
	KingPlayed           bool         `json:"kingPlayed"`
	BishopPlayed         bool         `json:"bishopPlayed"`
	KnightPlayed         bool         `json:"knightPlayed"`
	RookPlayed           bool         `json:"rookPlayed"`
	NumPawns             int          `json:"numPawns"`
	ManaMax              int          `json:"manaMax"`
	ManaCurrent          int          `json:"manaCurrent"`
	KingHP               int          `json:"kingHP"`
	KingAttack           int          `json:"kingAttack"`
	BishopHP             int          `json:"bishopHP"`
	BishopAttack         int          `json:"bishopAttack"`
	KnightHP             int          `json:"knightHP"`
	KnightAttack         int          `json:"knightAttack"`
	RookHP               int          `json:"rookHP"`
	RookAttack           int          `json:"rookAttack"`
	Color                string       `json:"color"`
	Other                *PublicState `json:"-"` // convenient way of getting opponent
}

type Piece struct {
	Name   string `json:"name"`
	Color  string `json:"color"`
	HP     int    `json:"hp"`
	Attack int    `json:"attack"`
	Damage int    `json:"damage"` // amount of damage unit will take in combat
}

type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Card struct {
	Name     string `json:"name"`
	ManaCost int    `json:"manaCost"`
}

type MatchMap struct {
	sync.RWMutex
	internal map[string]*Match
}
