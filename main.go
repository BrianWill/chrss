package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/heroku/x/hmetrics/onload"
	uuid "github.com/satori/go.uuid"
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
	pawnHP       = 3
	pawnAttack   = 2
	kingHP       = 40
	kingAttack   = 15
	bishopHP     = 15
	bishopAttack = 8
	knightHP     = 15
	knightAttack = 5
	rookHP       = 20
	rookAttack   = 10
)

const defaultInstruction = "Pick a card to play or pass."
const kingInstruction = "Pick a square to place your king."

const nColumns = 6
const nRows = 6

const turnTimer = 50 * int64(time.Second)

type Phase string

const (
	readyUpPhase       Phase = "readyUp"
	mainPhase          Phase = "main"
	kingPlacementPhase Phase = "kingPlacement"
	reclaimPhase       Phase = "reclaim"
	gameoverPhase      Phase = "gameover"
)

var positions [nColumns * nRows]Pos // convenience for getting Pos of board index

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const maxReclaim = 2 // max number of pieces to reclaim at end of round
const matchTimeout = 120 * int64(time.Minute)

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
	Cards             []Card `json:"cards"`
	SelectedCard      int    `json:"selectedCard"` // index into cards slice
	SelectedPos       Pos    `json:"selectedPos"`
	HighlightEmpty    bool   `json:"highlightEmpty"` // highlight the empty squares on the player's side
	PlayerInstruction string `json:"playerInstruction"`
	ReclaimSelections []Pos  `json:"reclaimSelections"`
}

type PublicState struct {
	Ready                bool `json:"ready"` // match does not start until both player's are ready
	ReclaimSelectionMade bool `json:"reclaimSelectionMade"`
	KingPlayed           bool `json:"kingPlayed"`
	BishopPlayed         bool `json:"bishopPlayed"`
	KnightPlayed         bool `json:"knightPlayed"`
	RookPlayed           bool `json:"rookPlayed"`
	NumPawns             int  `json:"numPawns"`
	ManaMax              int  `json:"manaMax"`
	ManaCurrent          int  `json:"manaCurrent"`
	KingHP               int  `json:"kingHP"`
	KingAttack           int  `json:"kingAttack"`
	BishopHP             int  `json:"bishopHP"`
	BishopAttack         int  `json:"bishopAttack"`
	KnightHP             int  `json:"knightHP"`
	KnightAttack         int  `json:"knightAttack"`
	RookHP               int  `json:"rookHP"`
	RookAttack           int  `json:"rookAttack"`
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
	Owner    string `json:"owner"` // black, white, none
	ManaCost int    `json:"manaCost"`
}

type MatchMap struct {
	sync.RWMutex
	internal map[string]*Match
}

func drawCards(owner string, existing []Card, public *PublicState) []Card {
	// remove stock from existing
	i := 0
loop:
	for ; i < len(existing); i++ {
		switch existing[i].Name {
		case king, bishop, knight, rook:
		default:
			break loop
		}
	}
	existing = existing[i:]

	stock := []Card{}
	if public.BishopHP > 0 && !public.BishopPlayed {
		stock = append(stock, Card{bishop, owner, 0})
	}
	if public.KnightHP > 0 && !public.KnightPlayed {
		stock = append(stock, Card{knight, owner, 0})
	}
	if public.RookHP > 0 && !public.RookPlayed {
		stock = append(stock, Card{rook, owner, 0})
	}

	return append(stock, existing...)
}

func drawCommunalCards() []Card {
	return []Card{}
}

func (m *Match) IsOpen() bool {
	return (m.WhiteConn == nil || m.BlackConn == nil) && m.Winner == none
}

func (m *Match) IsBlackOpen() bool {
	return m.BlackConn == nil && m.Winner == none
}

func (m *Match) IsWhiteOpen() bool {
	return m.WhiteConn == nil && m.Winner == none
}

func (m *Match) IsFull() bool {
	return (m.WhiteConn != nil && m.BlackConn != nil) && m.Winner == none
}

func (m *Match) IsFinished() bool {
	return m.Winner != none
}

// panics if outo f bounds
func (p *PrivateState) RemoveCard(idx int) {
	p.Cards = append(p.Cards[:idx], p.Cards[idx+1:]...)
}

// get n random values from slice (mutates input slice)
// (shuffles whole slice, so not ideal for large slice)
func randSelect(n int, candidates []int) []int {
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	if n > len(candidates) {
		return candidates
	}
	return candidates[:n]
}

func initMatch(m *Match) {
	// random adjective-animal
	m.Name = adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
	m.LastMoveTime = time.Now().UnixNano()
	m.StartTime = m.LastMoveTime
	m.Turn = white
	m.Winner = none
	m.Round = 0 // will get incremented to 1 once both players ready up
	m.Phase = readyUpPhase

	m.WhitePublic.ManaCurrent = 3
	m.WhitePublic.ManaMax = 3
	m.WhitePublic.KingHP = kingHP
	m.WhitePublic.KingAttack = kingAttack
	m.WhitePublic.BishopHP = bishopHP
	m.WhitePublic.BishopAttack = bishopAttack
	m.WhitePublic.KnightHP = knightHP
	m.WhitePublic.KnightAttack = knightAttack
	m.WhitePublic.RookHP = rookHP
	m.WhitePublic.RookAttack = rookAttack

	m.BlackPublic.ManaCurrent = 3
	m.BlackPublic.ManaMax = 3
	m.BlackPublic.KingHP = kingHP
	m.BlackPublic.KingAttack = kingAttack
	m.BlackPublic.BishopHP = bishopHP
	m.BlackPublic.BishopAttack = bishopAttack
	m.BlackPublic.KnightHP = knightHP
	m.BlackPublic.KnightAttack = knightAttack
	m.BlackPublic.RookHP = rookHP
	m.BlackPublic.RookAttack = rookAttack

	m.SpawnPawns(white, true)
	m.SpawnPawns(black, true)
	m.CalculateDamage()

	m.BlackPrivate = PrivateState{
		Cards:             drawCards(black, nil, &m.BlackPublic),
		SelectedCard:      -1,
		SelectedPos:       Pos{-1, -1},
		PlayerInstruction: defaultInstruction,
		HighlightEmpty:    true,
	}
	// white starts ready to play king
	m.WhitePrivate = PrivateState{
		Cards:             drawCards(white, nil, &m.WhitePublic),
		SelectedCard:      -1,
		SelectedPos:       Pos{-1, -1},
		PlayerInstruction: defaultInstruction,
		HighlightEmpty:    true,
	}
}

// returns nil for empty square
func (m *Match) getPiece(p Pos) *Piece {
	return m.Board[nColumns*p.Y+p.X]
}

// does not panic
func (m *Match) getPieceSafe(p Pos) *Piece {
	if p.X < 0 || p.X >= nColumns || p.Y < 0 || p.Y >= nRows {
		return nil
	}
	return m.Board[nColumns*p.Y+p.X]
}

// panics if out of bounds
func (m *Match) setPiece(p Pos, piece Piece) {
	idx := nColumns*p.Y + p.X
	m.pieces[idx] = piece
	m.Board[idx] = &m.pieces[idx]
}

// panics if out of bounds
func (m *Match) removePieceAt(p Pos) {
	idx := nColumns*p.Y + p.X
	m.Board[idx] = nil
	m.pieces[idx] = Piece{}
}

func (m *Match) RemoveNonPawns() {
	for i, p := range m.pieces {
		if p.Name != pawn {
			m.pieces[i] = Piece{}
			m.Board[i] = nil
		}
	}
}

func (m *Match) InflictDamage() {
	for i, p := range m.Board {
		if p != nil {
			p.HP -= p.Damage
			p.Damage = 0
			var public *PublicState
			if p.Color == black {
				public = &m.BlackPublic
			} else {
				public = &m.WhitePublic
			}
			switch p.Name {
			case king:
				public.KingHP = p.HP
			case bishop:
				public.BishopHP = p.HP
			case knight:
				public.KnightHP = p.HP
			case rook:
				public.RookHP = p.HP
			}
			if p.HP <= 0 {
				if p.Name == pawn {
					public.NumPawns--
				}
				m.pieces[i] = Piece{}
				m.Board[i] = nil
			}
		}
	}
}

func (m *Match) CalculateDamage() {
	// todo
	rookAttack := func(p Pos, color string, attack int) {
		x := p.X + 1
		y := p.Y
		for x < nColumns {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
		}

		x = p.X - 1
		y = p.Y
		for x >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
		}

		x = p.X
		y = p.Y + 1
		for y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			y++
		}

		x = p.X
		y = p.Y - 1
		for y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			y--
		}
	}

	bishopAttack := func(p Pos, color string, attack int) {
		x := p.X + 1
		y := p.Y + 1
		for x < nColumns && y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
			y++
		}

		x = p.X - 1
		y = p.Y + 1
		for x >= 0 && y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
			y++
		}

		x = p.X + 1
		y = p.Y - 1
		for x < nColumns && y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
			y--
		}

		x = p.X - 1
		y = p.Y - 1
		for x >= 0 && y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
			y--
		}
	}

	kingAttack := func(p Pos, color string, attack int) {
		ps := []Pos{
			Pos{p.X + 1, p.Y + 1},
			Pos{p.X + 1, p.Y},
			Pos{p.X + 1, p.Y - 1},
			Pos{p.X, p.Y + 1},
			Pos{p.X, p.Y - 1},
			Pos{p.X - 1, p.Y + 1},
			Pos{p.X - 1, p.Y},
			Pos{p.X - 1, p.Y - 1},
		}
		for _, other := range ps {
			hit := m.getPieceSafe(other)
			if hit != nil && hit.Color != color {
				hit.Damage += attack
			}
		}
	}

	knightAttack := func(p Pos, color string, attack int) {
		ps := []Pos{
			Pos{p.X + 1, p.Y + 2},
			Pos{p.X + 1, p.Y - 2},
			Pos{p.X + 2, p.Y + 1},
			Pos{p.X + 2, p.Y - 1},
			Pos{p.X - 1, p.Y + 2},
			Pos{p.X - 1, p.Y - 2},
			Pos{p.X - 2, p.Y + 1},
			Pos{p.X - 2, p.Y - 1},
		}
		for _, other := range ps {
			hit := m.getPieceSafe(other)
			if hit != nil && hit.Color != color {
				hit.Damage += attack
			}
		}
	}

	pawnAttack := func(p Pos, color string, attack int) {
		yOffset := 1
		if color == black {
			yOffset = -1
		}
		ps := []Pos{
			Pos{p.X + 1, p.Y + yOffset},
			Pos{p.X - 1, p.Y + yOffset},
		}
		for _, other := range ps {
			hit := m.getPieceSafe(other)
			if hit != nil && hit.Color != color {
				hit.Damage += attack
			}
		}
	}

	// reset all to 0
	for i := range m.pieces {
		m.pieces[i].Damage = 0
	}

	attackMap := map[string]func(Pos, string, int){
		king:   kingAttack,
		bishop: bishopAttack,
		knight: knightAttack,
		rook:   rookAttack,
		pawn:   pawnAttack,
	}

	// visit each piece, adding the damage it inflicts on other pieces
	for i, p := range m.Board {
		if p != nil {
			attackMap[p.Name](positions[i], p.Color, p.Attack)
		}
	}
}

// spawn n random pawns in free columns
func (m *Match) SpawnPawns(player string, init bool) {
	whiteNumPawns := m.WhitePublic.NumPawns
	blackNumPawns := m.BlackPublic.NumPawns
	n := 1
	if init {
		n = 4
	}
	var columns []int
	if player == white {
		if !init && whiteNumPawns == 0 {
			n = 2
		} else if whiteNumPawns == 5 {
			return // keep max pawns at 5
		}
		for i := 0; i < nColumns; i++ {
			if m.getPiece(Pos{i, 1}) == nil && m.getPiece(Pos{i, 2}) == nil {
				columns = append(columns, i)
			}
		}
		columns = randSelect(n, columns)
		for _, v := range columns {
			m.setPiece(Pos{v, rand.Intn(2) + 1}, Piece{pawn, white, pawnHP, pawnAttack, 0})
		}
		m.WhitePublic.NumPawns = whiteNumPawns + n
	} else {
		if !init && blackNumPawns == 0 {
			n = 2
		} else if blackNumPawns == 5 {
			return // keep max pawns at 5
		}
		for i := 0; i < nColumns; i++ {
			if m.getPiece(Pos{i, 3}) == nil && m.getPiece(Pos{i, 4}) == nil {
				columns = append(columns, i)
			}
		}
		columns = randSelect(n, columns)
		for _, v := range columns {
			m.setPiece(Pos{v, rand.Intn(2) + 3}, Piece{pawn, black, pawnHP, pawnAttack, 0})
		}
		m.BlackPublic.NumPawns = blackNumPawns + n
	}
}

// returns boolean true when no free slot
func (m *Match) RandomFreeSquare(player string) (Pos, bool) {
	// collect Pos of all free squares on player's side
	freeSquares := []Pos{}
	x := 0
	y := 0
	i := 0
	end := len(m.Board) / 2
	if player == black {
		y = nRows / 2
		i = end
		end = len(m.Board)
	}
	for ; i < end; i++ {
		if m.Board[i] == nil {
			freeSquares = append(freeSquares, Pos{x, y})
		}
		x++
		if x == nColumns {
			x = 0
			y++
		}
	}
	if len(freeSquares) == 0 {
		return Pos{}, true
	}

	// random pick from the free Pos
	return freeSquares[rand.Intn(len(freeSquares))], false
}

func (m *Match) states(color string) (*PublicState, *PrivateState) {
	if color == black {
		return &m.BlackPublic, &m.BlackPrivate
	} else {
		return &m.WhitePublic, &m.WhitePrivate
	}
}

const reclaimHealRook = 5

func (m *Match) ReclaimPieces() {
	for _, color := range []string{white, black} {
		public, private := m.states(color)
		if public.ReclaimSelectionMade {
			for i, pos := range private.ReclaimSelections {
				if i >= 2 {
					break // in case more than two selections were sent, we ignore all but first two
				}
				p := m.getPieceSafe(pos)
				if p != nil { // selection might be off the board or a square without a piece (todo: send error to client)
					// if reclaiming a king or vassal, add card back to hand
					// if reclaiming a rook, heal the rook
					switch p.Name {
					case king:
						m.removePieceAt(pos)
						public.KingPlayed = false
					case bishop:
						m.removePieceAt(pos)
						public.BishopPlayed = false
					case knight:
						m.removePieceAt(pos)
						public.KnightPlayed = false
					case rook:
						m.removePieceAt(pos)
						public.RookPlayed = false
					case pawn:
						m.removePieceAt(pos)
					default:

					}
				}
			}
		}
		private.ReclaimSelections = nil
		public.ReclaimSelectionMade = false
	}
}

func (m *Match) EndRound() {
	m.LastMoveTime = time.Now().UnixNano()
	m.Round++
	m.Log = append(m.Log, "Round "+strconv.Itoa(m.Round))

	if m.FirstTurnColor == black {
		m.Turn = white
		m.FirstTurnColor = white
	} else {
		m.Turn = black
		m.FirstTurnColor = black
	}

	m.BlackPublic.ManaMax++
	m.BlackPublic.ManaCurrent = m.BlackPublic.ManaMax

	m.WhitePublic.ManaMax++
	m.WhitePublic.ManaCurrent = m.WhitePublic.ManaMax

	m.PassPrior = false

	m.WhitePrivate.Cards = drawCards(white, m.WhitePrivate.Cards, &m.WhitePublic)
	m.BlackPrivate.Cards = drawCards(black, m.BlackPrivate.Cards, &m.BlackPublic)
	m.WhitePrivate.SelectedCard = -1
	m.BlackPrivate.SelectedCard = -1

	m.SpawnPawns(black, false)
	m.SpawnPawns(white, false)
	m.CalculateDamage()

	if m.WhitePublic.KingPlayed && m.BlackPublic.KingPlayed {
		m.Phase = mainPhase
		m.BlackPrivate.HighlightEmpty = false
		m.WhitePrivate.HighlightEmpty = false
	} else {
		m.Phase = kingPlacementPhase
		m.BlackPrivate.HighlightEmpty = true
		m.WhitePrivate.HighlightEmpty = true
	}
}

func (m *Match) EndKingPlacement() bool {
	if m.WhitePublic.KingPlayed && m.BlackPublic.KingPlayed {
		m.CalculateDamage()
		m.LastMoveTime = time.Now().UnixNano()
		m.WhitePrivate.HighlightEmpty = false
		m.BlackPrivate.HighlightEmpty = false
		m.Phase = mainPhase
		return true
	}
	return false
}

// pass = if turn is ending by passing; player = color whose turn is ending
func (m *Match) EndTurn(pass bool, player string) {
	m.LastMoveTime = time.Now().UnixNano()
	m.CalculateDamage()

	if pass && m.PassPrior { // if players both pass in succession, do combat
		m.InflictDamage()

		blackVassalsDead := m.BlackPublic.BishopHP <= 0 && m.BlackPublic.KnightHP <= 0 && m.BlackPublic.RookHP <= 0
		whiteVassalsDead := m.WhitePublic.BishopHP <= 0 && m.WhitePublic.KnightHP <= 0 && m.WhitePublic.RookHP <= 0
		blackKingDead := m.BlackPublic.KingHP <= 0
		whiteKingDead := m.WhitePublic.KingHP <= 0

		if (blackKingDead || blackVassalsDead) && (whiteKingDead || whiteVassalsDead) {
			m.Winner = draw
			m.Phase = gameoverPhase
		} else if blackKingDead || blackVassalsDead {
			m.Winner = white
			m.Phase = gameoverPhase
		} else if whiteKingDead || whiteVassalsDead {
			m.Winner = black
			m.Phase = gameoverPhase
		} else {
			m.Phase = reclaimPhase
		}
	} else {
		if m.Turn == black {
			m.Turn = white
		} else {
			m.Turn = black
		}
		m.PassPrior = pass

		m.WhitePrivate.PlayerInstruction = defaultInstruction
		m.WhitePrivate.SelectedCard = -1
		m.WhitePrivate.HighlightEmpty = false

		m.BlackPrivate.PlayerInstruction = defaultInstruction
		m.BlackPrivate.SelectedCard = -1
		m.BlackPrivate.HighlightEmpty = false
	}
}

func processMessage(msg []byte, match *Match, player string) {
	currentRound := match.Round
	newTurn := false
	var event string
	var notifyOpponent bool // set to true for events where opponent should get state update
	idx := 0
	for ; idx < len(msg); idx++ {
		if msg[idx] == ' ' {
			event = string(msg[:idx])
			msg = msg[idx+1:]
		}
	}
	//fmt.Println("event ", event, string(msg))
	match.Mutex.Lock()
	if match.Phase != gameoverPhase {
		public, private := match.states(player)
		switch event {
		case "ping":
			match.Mutex.Unlock()
			return
		case "get_state":
			// doesn't change anything, just fetches current state
		case "ready":
			switch match.Phase {
			case readyUpPhase:
				public.Ready = true
				if match.BlackPublic.Ready && match.WhitePublic.Ready {
					match.Phase = kingPlacementPhase
					match.Round = 1 // by incrementing from 0, will sound new round fanfare
					match.Log = []string{"Round 1"}
					match.LastMoveTime = time.Now().UnixNano()
				}
				notifyOpponent = true
			}
		case "reclaim_time_expired":
			switch match.Phase {
			case reclaimPhase:
				turnElapsed := time.Now().UnixNano() - match.LastMoveTime
				remainingTurnTime := turnTimer - turnElapsed
				if remainingTurnTime > 0 {
					break // ignore if time hasn't actually expired
				}
				match.ReclaimPieces()
				match.EndRound()
				notifyOpponent = true
			}
		case "reclaim_done":
			switch match.Phase {
			case reclaimPhase:
				public.ReclaimSelectionMade = true
				// if other player already has a reclaim selection, then we move on to next round
				if (player == black && match.WhitePublic.ReclaimSelectionMade) ||
					(player == white && match.BlackPublic.ReclaimSelectionMade) {
					match.ReclaimPieces()
					match.EndRound()
					notifyOpponent = true
				} else {
					notifyOpponent = true
				}
			}
		case "time_expired":
			switch match.Phase {
			case mainPhase:
				turnElapsed := time.Now().UnixNano() - match.LastMoveTime
				remainingTurnTime := turnTimer - turnElapsed
				if remainingTurnTime > 0 {
					break // ignore if time hasn't actually expired
				}
				// actual elapsed time is checked on server, but we rely upon clients to notify
				// (not ideal because both clients might fail, but then we have bigger problem)
				// Cheater could supress sending time_expired event from their client, but
				// opponent also sends the event (and has interest to do so).
				match.Log = append(match.Log, match.Turn+" passed")
				match.EndTurn(true, match.Turn)
				newTurn = true
				notifyOpponent = true
			case kingPlacementPhase:
				turnElapsed := time.Now().UnixNano() - match.LastMoveTime
				remainingTurnTime := turnTimer - turnElapsed
				if remainingTurnTime > 0 {
					break // ignore if time hasn't actually expired
				}
				for _, color := range []string{black, white} {
					public, _ := match.states(color)
					if !public.KingPlayed {
						// randomly place king in free square
						// because we must have reclaimed the King, there will always be a free square at this point
						pos, _ := match.RandomFreeSquare(color)
						match.setPiece(pos, Piece{king, color, public.KingHP, public.KingAttack, 0})
						public.KingPlayed = true
						match.Log = append(match.Log, color+" played King")
					}
				}
				newTurn = match.EndKingPlacement()
				notifyOpponent = true
			}
		case "click_card":
			switch match.Phase {
			case mainPhase:
				if player != match.Turn {
					break // ignore if not the player's turn
				}
				if !public.KingPlayed {
					break // cannot select other cards until king is played
				}
				type ClickCardEvent struct {
					SelectedCard int
				}
				var event ClickCardEvent
				err := json.Unmarshal(msg, &event)
				if err != nil {
					fmt.Println("unmarssalling click_card error", err)
					break // todo: send error response
				}
				if event.SelectedCard == private.SelectedCard {
					private.SelectedCard = -1
					private.HighlightEmpty = false
					private.PlayerInstruction = defaultInstruction
				} else {
					private.SelectedCard = event.SelectedCard
					private.HighlightEmpty = true
					private.PlayerInstruction = "Click an empty spot on your side of the board to place the card."
				}
			}
		case "click_board":
			type ClickBoardEvent struct {
				X int
				Y int
			}
			var event ClickBoardEvent
			err := json.Unmarshal(msg, &event)
			if err != nil {
				break // todo: send error response
			}
			switch match.Phase {
			case mainPhase:
				if player != match.Turn {
					break // ignore if not the player's turn
				}
				// ignore if not card selected
				if private.SelectedCard == -1 {
					break
				}
				// ignore clicks on occupied spaces
				if match.getPieceSafe(Pos{event.X, event.Y}) != nil {
					break
				}
				// square must be on player's side of board
				if player == white && event.Y >= nColumns/2 {
					break
				}
				if player == black && event.Y < nColumns/2 {
					break
				}

				card := private.Cards[private.SelectedCard]
				var p Piece
				switch card.Name {
				case bishop:
					p = Piece{bishop, player, public.BishopHP, public.BishopAttack, 0}
					public.BishopPlayed = true
				case knight:
					p = Piece{knight, player, public.KnightHP, public.KnightAttack, 0}
					public.KnightPlayed = true
				case rook:
					p = Piece{rook, player, public.RookHP, public.RookAttack, 0}
					public.RookPlayed = true
				}
				match.Log = append(match.Log, player+" played "+card.Name)
				match.setPiece(Pos{event.X, event.Y}, p)
				// remove card
				private.RemoveCard(private.SelectedCard)
				match.EndTurn(false, player)
				newTurn = true
				notifyOpponent = true
			case reclaimPhase:
				pos := Pos{event.X, event.Y}
				p := match.getPieceSafe(pos)
				if p != nil && p.Color == player {
					found := false
					selections := private.ReclaimSelections
					for i, selection := range selections {
						if selection == pos {
							selections = append(selections[:i], selections[i+1:]...)
							found = true
						}
					}
					if !found && len(selections) < maxReclaim {
						selections = append(selections, pos)
					}
					private.ReclaimSelections = selections
				}
			case kingPlacementPhase:
				if public.KingPlayed {
					break
				}
				// ignore clicks on occupied spaces
				if match.getPieceSafe(Pos{event.X, event.Y}) != nil {
					break
				}
				// square must be on player's side of board
				if player == white && event.Y >= nColumns/2 {
					break
				}
				if player == black && event.Y < nColumns/2 {
					break
				}
				public.KingPlayed = true
				match.Log = append(match.Log, player+" played King")
				match.setPiece(Pos{event.X, event.Y}, Piece{king, player, public.KingHP, public.KingAttack, 0})
				newTurn = match.EndKingPlacement()
				notifyOpponent = true
			}
		case "pass":
			switch match.Phase {
			case mainPhase:
				if player != match.Turn {
					break // ignore if not the player's turn
				}
				if !public.KingPlayed {
					break // cannot pass when king has not been played
				}
				match.Log = append(match.Log, player+" passed")
				match.EndTurn(true, player)
				newTurn = true
				notifyOpponent = true
			}
		default:
			fmt.Println("bad event: ", event, msg) // todo: better error reporting
		}
	}
	processConnection := func(conn *websocket.Conn, color string, private *PrivateState, newTurn bool) {
		turnElapsed := time.Now().UnixNano() - match.LastMoveTime
		remainingTurnTime := (turnTimer - turnElapsed) / 1000000
		if conn != nil {
			response := gin.H{
				"turnRemainingMilliseconds": remainingTurnTime,
				"color":                     color,
				"board":                     match.Board,
				"private":                   private,
				"turn":                      match.Turn,
				"newTurn":                   newTurn,
				"winner":                    match.Winner,
				"round":                     match.Round,
				"newRound":                  match.Round > currentRound,
				"lastMoveTime":              match.LastMoveTime,
				"blackPublic":               match.BlackPublic,
				"whitePublic":               match.WhitePublic,
				"passPrior":                 match.PassPrior,
				"phase":                     match.Phase,
				"firstTurnColor":            match.FirstTurnColor,
				"log":                       match.Log,
			}
			bytes, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("Error JSON encoding state: %+v", err)
			}
			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				if !websocket.IsCloseError(err) {
					fmt.Printf("Error writing message to %+v connection: %+v", color, err)
				}
			}
		}
	}
	if player == black {
		processConnection(match.BlackConn, black, &match.BlackPrivate, newTurn)
		if notifyOpponent {
			processConnection(match.WhiteConn, white, &match.WhitePrivate, newTurn)
		}
	} else {
		processConnection(match.WhiteConn, white, &match.WhitePrivate, newTurn)
		if notifyOpponent {
			processConnection(match.BlackConn, black, &match.BlackPrivate, newTurn)
		}
	}

	match.Mutex.Unlock()
}

func NewMatchMap() *MatchMap {
	return &MatchMap{
		internal: make(map[string]*Match),
	}
}

func (mm *MatchMap) Load(key string) (value *Match, ok bool) {
	mm.RLock()
	result, ok := mm.internal[key]
	mm.RUnlock()
	return result, ok
}

func (mm *MatchMap) Delete(key string) {
	mm.Lock()
	delete(mm.internal, key)
	mm.Unlock()
}

func (mm *MatchMap) Store(key string, value *Match) {
	mm.Lock()
	mm.internal[key] = value
	mm.Unlock()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	liveMatches := NewMatchMap()
	{
		x := 0
		y := 0
		for i := range positions {
			positions[i] = Pos{x, y}
			x++
			if x == nColumns {
				x = 0
				y++
			}
		}
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/static", "static")

	// list open matches
	router.GET("/", func(c *gin.Context) {
		type match struct {
			Name      string
			UUID      string
			BlackOpen bool
			WhiteOpen bool
		}
		type matchList struct {
			Open []match
			Full []match
		}
		liveMatches.Lock()
		matches := matchList{}
		i := 0
		for _, m := range liveMatches.internal {
			black, white := m.IsBlackOpen(), m.IsWhiteOpen()
			if black || white {
				matches.Open = append(matches.Open, match{m.Name, m.UUID, black, white})
			} else {
				matches.Full = append(matches.Full, match{m.Name, m.UUID, black, white})
			}
			i++
		}
		sort.Slice(matches.Open, func(i, j int) bool { return matches.Open[i].Name < matches.Open[j].Name })
		sort.Slice(matches.Full, func(i, j int) bool { return matches.Full[i].Name < matches.Full[j].Name })
		liveMatches.Unlock()

		c.HTML(http.StatusOK, "browse.tmpl", matches)
	})

	router.GET("/guide", func(c *gin.Context) {
		c.HTML(http.StatusOK, "guide.tmpl", nil)
	})

	// periodically clean liveMatches of finished or timedout games
	go func() {
		for {
			liveMatches.Lock()
			for id, match := range liveMatches.internal {
				exceededTimeout := time.Now().Unix() > match.LastMoveTime+matchTimeout
				if match.Phase == gameoverPhase || exceededTimeout {
					delete(liveMatches.internal, id)
				}
			}
			liveMatches.Unlock()
			time.Sleep(5 * time.Minute)
		}
	}()

	router.GET("/createMatch", func(c *gin.Context) {
		u4, err := uuid.NewV4()
		if err != nil {
			c.String(http.StatusInternalServerError, "Could not generate UUIDv4: %v", err)
			return
		}
		u4Str := u4.String()
		match := &Match{
			UUID: u4Str,
		}
		initMatch(match)
		liveMatches.Store(u4Str, match)
		var color string
		if rand.Intn(2) == 0 {
			color = black
		} else {
			color = white
		}
		c.Redirect(http.StatusSeeOther, "/match/"+u4Str+"/"+color)
	})

	// pass in UUID and optionally a password (from cookie? get param?)
	router.GET("/match/:id/:color", func(c *gin.Context) {
		id := c.Param("id")
		color := c.Param("color")
		if color != "black" && color != "white" {
			c.String(http.StatusNotFound, "Must specify black or white. Invalid match color: '%s'.", color)
			return
		}
		log.Printf("joining match: %v\n", id)
		if _, ok := liveMatches.Load(id); !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", id)
			return
		}
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	router.GET("/ws/:id/:color", func(c *gin.Context) {
		id := c.Param("id")
		color := c.Param("color")
		log.Printf("making match connection: " + id + " " + color)
		if color != "black" && color != "white" {
			c.String(http.StatusNotFound, "Must specify black or white. Invalid match color: '%s'.", color)
			return
		}
		match, ok := liveMatches.Load(id)
		if !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", id)
			return
		}

		match.Mutex.Lock()
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Printf("Failed to set websocket upgrade: %+v", err)
			return
		}

		if (color == black && match.BlackConn != nil) || (color == white && match.WhiteConn != nil) {
			response := gin.H{
				"error": "This match already has a player for '" + color + "'.",
			}
			bytes, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("Error JSON encoding state: %+v", err)
			}
			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				fmt.Printf("Error sending 'match full' message: %+v", err)
			}
			match.Mutex.Unlock()
			goto exit
		} else if color == black {
			match.BlackConn = conn
		} else if color == white {
			match.WhiteConn = conn
		}

		match.Mutex.Unlock()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			processMessage(msg, match, color)
		}

	exit:
		conn.Close()
		match.Mutex.Lock()
		if color == black {
			match.BlackConn = nil
		} else if color == white {
			match.WhiteConn = nil
		}
		fmt.Printf("Closed connection '%s' in match %s %s", color, match.Name, match.UUID)
		match.Mutex.Unlock()
	})

	router.Run(":" + port)
}
