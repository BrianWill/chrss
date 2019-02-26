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
	bishopMana   = 1
	knightHP     = 25
	knightAttack = 5
	knightMana   = 1
	rookHP       = 20
	rookAttack   = 6
	rookMana     = 2
	queenHP      = 15
	queenAttack  = 6
	queenMana    = 3
	jesterHP     = 12
	jesterAttack = 0
	jesterMana   = 3
)

const (
	castleCard               = "Castle"
	castleMana               = 2
	reclaimVassalCard        = "Reclaim Vassal"
	reclaimVassalMana        = 2
	swapFrontLinesCard       = "Swap Front Lines"
	swapFrontLinesMana       = 2
	removePawnCard           = "Remove Pawn"
	removePawnMana           = 2
	forceCombatCard          = "Force Combat"
	forceCombatMana          = 3
	mirrorCard               = "Mirror"
	mirrorMana               = 2
	healCard                 = "Heal"
	healMana                 = 2
	healCardAmount           = 5
	drainManaCard            = "Drain Mana"
	drainManaMana            = 2
	drainManaAmount          = 2
	togglePawnCard           = "Toggle Pawn"
	togglePawnMana           = 1
	nukeCard                 = "Nuke"
	nukeMana                 = 2
	nukeDamageFull           = 6
	nukeDamageLesser         = 3
	shoveCard                = "Shove"
	shoveMana                = 2
	advanceCard              = "Advance"
	advanceMana              = 2
	restoreManaCard          = "Restore Mana"
	restoreManaMana          = 2
	summonPawnCard           = "Summon Pawn"
	summonPawnMana           = 2
	vulnerabilityCard        = "Vulnerability"
	vulnerabilityMana        = 2
	vulnerabilityFactor      = 2
	vulnerabilityDuration    = 1
	amplifyCard              = "Amplify"
	amplifyMana              = 2
	amplifyFactor            = 2
	amplifyDuration          = 1
	enrageCard               = "Enrage"
	enrageMana               = 2
	enrageDuration           = 1
	dodgeCard                = "Dodge"
	dodgeMana                = 2
	resurrectVassalCard      = "Resurrect Vassal"
	resurrectVassalMana      = 2
	resurrectVassalRestoreHP = 5
	stunVassalCard           = "Stun Vassal"
	stunVassalMana           = 2
	stunVassalDuration       = 1
	transparencyCard         = "Transparency"
	transparencyMana         = 2
	transparencyDuration     = 1
	armorCard                = "Armor"
	armorMana                = 2
	armorAmount              = 2
	dispellCard              = "Dispell"
	dispellMana              = 2
	poisonCard               = "Poison"
	poisonMana               = 2
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
	nCardsFirstRound = 3
	nCardsPerRound   = 2
	nCardsCap        = 8
)

const (
	highlightOff = iota
	highlightOn
	highlightDim
)

var allCards = []Card{
	Card{queen, queenMana},
	Card{jester, jesterMana},
	Card{castleCard, castleMana},
	Card{reclaimVassalCard, reclaimVassalMana},
	Card{vulnerabilityCard, vulnerabilityMana},
	Card{amplifyCard, amplifyMana},
	Card{stunVassalCard, stunVassalMana},
	Card{armorCard, armorMana},
	Card{poisonCard, poisonMana},
	Card{dispellCard, dispellMana},
	Card{enrageCard, enrageMana},
	Card{dodgeCard, dodgeMana},
	Card{transparencyCard, transparencyMana},
	Card{swapFrontLinesCard, swapFrontLinesMana},
	Card{removePawnCard, removePawnMana},
	Card{forceCombatCard, forceCombatMana},
	Card{mirrorCard, mirrorMana},
	Card{healCard, healMana},
	Card{drainManaCard, drainManaMana},
	Card{togglePawnCard, togglePawnMana},
	Card{nukeCard, nukeMana},
	Card{shoveCard, shoveMana},
	Card{advanceCard, advanceMana},
	Card{restoreManaCard, restoreManaMana},
	Card{summonPawnCard, summonPawnMana},
	Card{resurrectVassalCard, resurrectVassalMana},
}

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
	Name          string // used to identify the match in browser
	BlackConn     *websocket.Conn
	WhiteConn     *websocket.Conn
	BlackPlayerID string
	WhitePlayerID string
	CreatorName   string
	Mutex         sync.RWMutex
	DevMode       bool
	// rows stored in order top-to-bottom, e.g. nColumns is index of leftmost square in second row
	// (*Pierce better for empty square when JSONifying; Board[i] points to pieces[i]
	// the array is here simply for memory locality)
	// white side is indexes 0 up to (nColumns*nRows)/2
	pieces               [nColumns * nRows]Piece        // zero value for empty square
	Board                [nColumns * nRows]*Piece       // nil for empty square
	tempPieces           [nColumns * nRows]Piece        // used for AI scoring (reusing these arrays avoids wasteful allocations)
	tempBoard            [nColumns * nRows]*Piece       // "
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
	PassPrior          bool   // true if prior move was a pass
	FirstTurnColor     string // color of player who had first turn this round
	Round              int    // starts at 1
	Winner             string // white, black, none, draw
	StartTime          int64  // unix time
	LastMoveTime       int64  // should be initialized to match start time
	Log                []string
	Phase              Phase
}

// info a player doesn't want opponent to see
type PrivateState struct {
	Cards             []Card                `json:"cards"`
	SelectedCard      int                   `json:"selectedCard"`  // index into cards slice
	PlayableCards     []bool                `json:"playableCards"` // parallel to Cards
	Highlights        [nColumns * nRows]int `json:"highlights"`
	ReclaimSelections []Pos                 `json:"reclaimSelections"`
	KingPos           *Pos                  `json:"kingPos"` // used in king placement (placed king is not revealed to opponent until main phase)
	Other             *PrivateState         `json:"-"`
}

// individual player state that is visible to all
type PublicState struct {
	Ready                bool         `json:"ready"` // match does not start until both player's are ready
	ReclaimSelectionMade bool         `json:"reclaimSelectionMade"`
	King                 *Piece       `json:"king"` // exposed to JSON so as to correctly display king stats in king placement
	Rook                 *Piece       `json:"rook"`
	Knight               *Piece       `json:"knight"`
	Bishop               *Piece       `json:"bishop"`
	KingPlayed           bool         `json:"kingPlayed"`
	BishopPlayed         bool         `json:"bishopPlayed"`
	KnightPlayed         bool         `json:"knightPlayed"`
	RookPlayed           bool         `json:"rookPlayed"`
	NumPawns             int          `json:"numPawns"`
	ManaMax              int          `json:"manaMax"`
	ManaCurrent          int          `json:"manaCurrent"`
	Color                string       `json:"color"`
	Other                *PublicState `json:"-"` // convenient way of getting opponent
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
	Name     string `json:"name"`
	ManaCost int    `json:"manaCost"`
}

type MatchMap struct {
	sync.RWMutex
	internal map[string]*Match
}

type UserMap struct {
	sync.RWMutex
	internal map[string]bool
}
