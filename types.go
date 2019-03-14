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
	jester = "Jester"
)

const (
	pawnHP       = 5
	pawnAttack   = 4
	kingHP       = 35
	kingAttack   = 12
	bishopHP     = 25
	bishopAttack = 4
	bishopRank   = 1
	knightHP     = 25
	knightAttack = 5
	knightRank   = 1
	rookHP       = 20
	rookAttack   = 6
	rookRank     = 2
	queenHP      = 12
	queenAttack  = 6
	queenRank    = 4
	jesterHP     = 12
	jesterAttack = 0
	jesterRank   = 4
)

const (
	castleCard               = "Castle"
	castleRank               = 2
	reclaimVassalCard        = "Reclaim Vassal"
	reclaimVassalRank        = 2
	swapFrontLinesCard       = "Swap Front Lines"
	swapFrontLinesRank       = 2
	removePawnCard           = "Remove Pawn"
	removePawnRank           = 1
	forceCombatCard          = "Force Combat"
	forceCombatRank          = 2
	mirrorCard               = "Mirror"
	mirrorRank               = 3
	healCard                 = "Heal"
	healRank                 = 1
	healCardAmount           = 5
	togglePawnCard           = "Toggle Pawn"
	togglePawnRank           = 1
	nukeCard                 = "Nuke"
	nukeRank                 = 2
	nukeDamageFull           = 6
	nukeDamageLesser         = 3
	shoveCard                = "Shove"
	shoveRank                = 1
	advanceCard              = "Advance"
	advanceRank              = 1
	summonPawnCard           = "Summon Pawn"
	summonPawnRank           = 2
	vulnerabilityCard        = "Vulnerability"
	vulnerabilityRank        = 1
	vulnerabilityFactor      = 2
	vulnerabilityDuration    = 1
	amplifyCard              = "Amplify"
	amplifyRank              = 1
	amplifyFactor            = 2
	amplifyDuration          = 1
	enrageCard               = "Enrage"
	enrageRank               = 1
	enrageDuration           = 1
	dodgeCard                = "Dodge"
	dodgeRank                = 1
	resurrectVassalCard      = "Resurrect Vassal"
	resurrectVassalRank      = 3
	resurrectVassalRestoreHP = 5
	stunVassalCard           = "Stun Vassal"
	stunVassalRank           = 2
	stunVassalDuration       = 1
	transparencyCard         = "Transparency"
	transparencyRank         = 2
	transparencyDuration     = 1
	armorCard                = "Armor"
	armorRank                = 1
	armorAmount              = 2
	dispellCard              = "Dispell"
	dispellRank              = 1
	poisonCard               = "Poison"
	poisonRank               = 3
	poisonAmount             = 2
)

const (
	maxPawns      = 5
	startingPawns = 4
)
const reclaimHealRook = 5

const defaultInstruction = "Pick a card to play or pass."
const kingInstruction = "Pick a square to place your king."

const nColumns = 6
const nRows = 6

const turnTimer = 50 * int64(time.Second)
const turnTimerDev = 50 * int64(time.Minute)
const maxConcurrentMatches = 100

const (
	nCommandCards = 3
	nSoldierCards = 3
)

const (
	highlightOff = iota
	highlightOn
	highlightDim
)

const (
	numTurns        = 4
	numVassalTurns  = 2
	numCommandTurns = 1
	numSoldierTurns = 1
)

const (
	vassalCard  = "vassal"
	soldierCard = "soldier"
	commandCard = "command"
)

// all cards excepting the vassals and King
var allCards = []Card{
	Card{queen, queenRank, soldierCard},
	Card{jester, jesterRank, soldierCard},
	Card{castleCard, castleRank, commandCard},
	Card{reclaimVassalCard, reclaimVassalRank, commandCard},
	Card{vulnerabilityCard, vulnerabilityRank, commandCard},
	Card{amplifyCard, amplifyRank, commandCard},
	Card{stunVassalCard, stunVassalRank, commandCard},
	Card{armorCard, armorRank, commandCard},
	Card{poisonCard, poisonRank, commandCard},
	Card{dispellCard, dispellRank, commandCard},
	Card{enrageCard, enrageRank, commandCard},
	Card{dodgeCard, dodgeRank, commandCard},
	Card{transparencyCard, transparencyRank, commandCard},
	Card{swapFrontLinesCard, swapFrontLinesRank, commandCard},
	Card{removePawnCard, removePawnRank, commandCard},
	Card{forceCombatCard, forceCombatRank, commandCard},
	Card{mirrorCard, mirrorRank, commandCard},
	Card{healCard, healRank, commandCard},
	Card{togglePawnCard, togglePawnRank, commandCard},
	Card{nukeCard, nukeRank, commandCard},
	Card{shoveCard, shoveRank, commandCard},
	Card{advanceCard, advanceRank, commandCard},
	Card{summonPawnCard, summonPawnRank, commandCard},
	Card{resurrectVassalCard, resurrectVassalRank, commandCard},
}

var soldierCards = []Card{
	Card{queen, queenRank, soldierCard},
	Card{jester, jesterRank, soldierCard},
}

// all cards excepting the vassals and King
var commandCards = []Card{
	Card{castleCard, castleRank, commandCard},
	Card{reclaimVassalCard, reclaimVassalRank, commandCard},
	Card{vulnerabilityCard, vulnerabilityRank, commandCard},
	Card{amplifyCard, amplifyRank, commandCard},
	Card{stunVassalCard, stunVassalRank, commandCard},
	Card{armorCard, armorRank, commandCard},
	Card{poisonCard, poisonRank, commandCard},
	Card{dispellCard, dispellRank, commandCard},
	Card{enrageCard, enrageRank, commandCard},
	Card{dodgeCard, dodgeRank, commandCard},
	Card{transparencyCard, transparencyRank, commandCard},
	Card{swapFrontLinesCard, swapFrontLinesRank, commandCard},
	Card{removePawnCard, removePawnRank, commandCard},
	Card{forceCombatCard, forceCombatRank, commandCard},
	Card{mirrorCard, mirrorRank, commandCard},
	Card{healCard, healRank, commandCard},
	Card{togglePawnCard, togglePawnRank, commandCard},
	Card{nukeCard, nukeRank, commandCard},
	Card{shoveCard, shoveRank, commandCard},
	Card{advanceCard, advanceRank, commandCard},
	Card{summonPawnCard, summonPawnRank, commandCard},
	Card{resurrectVassalCard, resurrectVassalRank, commandCard},
}

var cardRankCount []int // at index i, how many cards have rank i or lower

type Phase string

const (
	readyUpPhase       Phase = "readyUp"
	mainPhase          Phase = "main"
	kingPlacementPhase Phase = "kingPlacement"
	gameoverPhase      Phase = "gameover"
)

const matchTimeout = 20 * int64(time.Minute)

type Match struct {
	Name                 string // used to identify the match in browser
	BlackConn            *websocket.Conn
	WhiteConn            *websocket.Conn
	BlackPlayerID        string
	WhitePlayerID        string
	CreatorName          string
	Mutex                sync.RWMutex
	DevMode              bool
	Board                Board
	BoardTemp            Board                          // used for AI scoring
	SquareStatusesDirect [nColumns * nRows]SquareStatus // the status effects applied directly to squares
	// the status effects on squares from pieces combined with the effects applied directly to the squares
	// (should be recomputed any time pieces are placed/moved/killed)
	SquareStatuses     [nColumns * nRows]SquareStatus
	tempSquareStatuses [nColumns * nRows]SquareStatus // used for AI scoring
	TurnTimer          int64
	CommunalCards      []Card // card in pool shared by both players
	BlackPrivate       PrivateState
	WhitePrivate       PrivateState
	BlackPublic        PublicState
	WhitePublic        PublicState
	BlackAI            bool
	WhiteAI            bool
	Turn               string // white, black
	FirstTurnColor     string // color of player who had first turn this round
	MaxRank            int    // max rank card to draw
	Round              int    // starts at 1
	Winner             string // white, black, none, draw
	StartTime          int64  // unix time
	LastMoveTime       int64  // should be initialized to match start time
	Log                []string
	Phase              Phase
}

type Board struct {
	// rows stored in order top-to-bottom, e.g. nColumns is index of leftmost square in second row
	// (*Pierce better for empty square when JSONifying; Board[i] points to pieces[i]
	// the array is here simply for memory locality)
	// white side is indexes 0 up to (nColumns*nRows)/2
	PiecesActual [nColumns * nRows]Piece  // zero value for empty square
	Pieces       [nColumns * nRows]*Piece `json:"Pieces"` // nil for empty square
}

// info a player doesn't want opponent to see
type PrivateState struct {
	Cards         []Card                `json:"cards"`
	SelectedCard  int                   `json:"selectedCard"`  // index into cards slice
	PlayableCards []bool                `json:"playableCards"` // parallel to Cards
	Highlights    [nColumns * nRows]int `json:"highlights"`
	KingPos       *Pos                  `json:"kingPos"` // used in king placement (placed king is not revealed to opponent until main phase)
	Other         *PrivateState         `json:"-"`
}

// individual player state that is visible to all
type PublicState struct {
	Ready           bool         `json:"ready"` // match does not start until both player's are ready
	King            *Piece       `json:"king"`  // exposed to JSON so as to correctly display king stats in king placement
	Rook            *Piece       `json:"rook"`
	NumTurnsLeft    int          `json:"turns"`
	NumVassalTurns  int          `json:"vassalTurns"`
	NumCommandTurns int          `json:"commandTurns"`
	NumSoldierTurns int          `json:"soldierTurns"`
	Knight          *Piece       `json:"knight"`
	Bishop          *Piece       `json:"bishop"`
	KingPlayed      bool         `json:"kingPlayed"`
	BishopPlayed    bool         `json:"bishopPlayed"`
	KnightPlayed    bool         `json:"knightPlayed"`
	RookPlayed      bool         `json:"rookPlayed"`
	NumPawns        int          `json:"numPawns"`
	Color           string       `json:"color"`
	Other           *PublicState `json:"-"` // convenient way of getting opponent
}

type Piece struct {
	Name   string       `json:"name"`
	Color  string       `json:"color"`
	HP     int          `json:"hp"`
	Attack int          `json:"attack"`
	Damage int          `json:"damage"` // amount of damage unit will take in combat
	Status *PieceStatus `json:"status"`
}

// status effects applied to individual square
type SquareStatus struct {
	Negative *SquareNegativeStatus `json:"negative"`
	Positive *SquarePositiveStatus `json:"positive"`
}

type SquareNegativeStatus struct {
	Distracted bool `json:"distracted"`
}

type SquarePositiveStatus struct {
}

// status effects applied to individual pieces
type PieceStatus struct {
	Negative *PieceNegativeStatus `json:"negative"`
	Positive *PiecePositiveStatus `json:"positive"`
}

// int fields last for some number of rounds
type PieceNegativeStatus struct {
	Vulnerability int `json:"vulnerability"` // increase damage this piece takes
	Distracted    int `json:"distracted"`    // piece does not attack
	Unreclaimable int `json:"unreclaimable"` // piece cannot be reclaimed
	Enraged       int `json:"enraged"`       // piece hits allies as well as enemies
	Transparent   int `json:"transparent"`   // piece does not block attacks
	Poison        int `json:"poison"`        // number of HP to remove in every combat phase
}

type PiecePositiveStatus struct {
	Amplify      int `json:"amplify"`      // increase damage this piece inflicts
	DamageImmune int `json:"damageImmune"` // does not take damage
	Armor        int `json:"armor"`        // number of armor points (not the duration: armor is permanent)
}

type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Card struct {
	Name string `json:"name"`
	Rank int    `json:"rank"`
	Type string `json:"type"`
}

type MatchMap struct {
	sync.RWMutex
	internal map[string]*Match
}

type UserMap struct {
	sync.RWMutex
	internal map[string]bool
}
