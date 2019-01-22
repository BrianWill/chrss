package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/heroku/x/hmetrics/onload"
	uuid "github.com/satori/go.uuid"
)

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const matchTimeout time.Duration = 120 * time.Minute

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
	StartTime      time.Time
	UUID           string
	BlackPrivate   PrivateState
	WhitePrivate   PrivateState
	BlackPublic    PublicState
	WhitePublic    PublicState
	Turn           string    // white, black
	PassPrior      bool      // true if prior move was a pass
	FirstTurnColor string    // color of player who had first turn this round
	Round          int       // starts at 1
	Winner         string    // white, black, none, draw
	LastMoveTime   time.Time // should be initialized to match start time
}

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

const defaultInstruction = "Pick a card to play or pass."
const kingInstruction = "Pick a square to place your king."

const nColumns = 6
const nRows = 6

// info a player doesn't want opponent to see
type PrivateState struct {
	Cards             []Card `json:"cards"`
	SelectedCard      int    `json:"selectedCard"` // index into cards slice
	SelectedPos       Pos    `json:"selectedPos"`
	HighlightEmpty    bool   `json:"highlightEmpty"` // highlight the empty squares on the player's side
	PlayerInstruction string `json:"playerInstruction"`
}

type PublicState struct {
	KingPlayed  bool
	ManaMax     int
	ManaCurrent int
}

type Piece struct {
	Name  string `json:"name"`
	Color string `json:"color"`
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

func drawCards(owner string, existing []Card) []Card {
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

	stock := []Card{
		Card{king, owner, 0},
		Card{bishop, owner, 0},
		Card{knight, owner, 0},
		Card{rook, owner, 0},
	}
	return append(stock, existing...)
}

func drawCommunalCards() []Card {
	return []Card{}
}

func (m *Match) IsOpen() bool {
	return (m.WhiteConn == nil || m.BlackConn == nil) && m.Winner == none
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
func randSelect(n int, options []int) []int {
	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})
	return options[:n]
}

func initMatch(m *Match) {
	// random adjective-animal
	m.Name = adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
	m.LastMoveTime = time.Now()

	// random pawns in 4 columns each side, each in either second or third row
	columns := make([]int, nColumns)
	for i := 0; i < nColumns; i++ {
		columns[i] = i
	}
	columns = randSelect(4, columns)
	for i := 0; i < 4; i++ {
		m.setPiece(Pos{columns[i], rand.Intn(2) + 1}, Piece{pawn, white})
	}
	columns = make([]int, nColumns)
	for i := 0; i < nColumns; i++ {
		columns[i] = i
	}
	columns = randSelect(4, columns)
	for i := 0; i < 4; i++ {
		m.setPiece(Pos{columns[i], rand.Intn(2) + 3}, Piece{pawn, black})
	}

	m.WhitePublic.ManaCurrent = 3
	m.WhitePublic.ManaMax = 3
	m.BlackPublic.ManaCurrent = 3
	m.BlackPublic.ManaMax = 3
	m.Round = 1

	m.CommunalCards = drawCards(none, nil)
	m.BlackPrivate = PrivateState{
		Cards:             drawCards(black, nil),
		SelectedCard:      -1,
		SelectedPos:       Pos{-1, -1},
		PlayerInstruction: defaultInstruction,
	}
	// white starts ready to play king
	m.WhitePrivate = PrivateState{
		Cards:             drawCards(white, nil),
		SelectedCard:      0,
		SelectedPos:       Pos{-1, -1},
		PlayerInstruction: kingInstruction,
		HighlightEmpty:    true,
	}
	m.StartTime = time.Now()
	m.Turn = white
	m.Winner = none
	m.LastMoveTime = m.StartTime
}

// returns nil for empty square
func (m *Match) getPiece(p Pos) *Piece {
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

// pass = if turn is ending by passing; player = color whose turn is ending
func (m *Match) EndTurn(pass bool, player string) {
	if pass && m.PassPrior { // if players both pass in succession, end round
		// todo: resolve combat

		// remove all non-pawns from board, refresh card decks
		m.Round++
		if m.FirstTurnColor == black {
			m.Turn = white
			m.FirstTurnColor = white

			m.BlackPrivate.PlayerInstruction = defaultInstruction
			m.BlackPrivate.SelectedCard = -1
			m.BlackPrivate.HighlightEmpty = false

			m.WhitePrivate.PlayerInstruction = kingInstruction
			m.WhitePrivate.SelectedCard = 0
			m.WhitePrivate.HighlightEmpty = true
		} else {
			m.Turn = black
			m.FirstTurnColor = black

			m.BlackPrivate.PlayerInstruction = kingInstruction
			m.BlackPrivate.SelectedCard = 0
			m.BlackPrivate.HighlightEmpty = true

			m.WhitePrivate.PlayerInstruction = defaultInstruction
			m.WhitePrivate.SelectedCard = -1
			m.WhitePrivate.HighlightEmpty = false
		}
		m.BlackPublic.ManaMax++
		m.BlackPublic.ManaCurrent = m.BlackPublic.ManaMax
		m.BlackPublic.KingPlayed = false

		m.WhitePublic.ManaMax++
		m.WhitePublic.ManaCurrent = m.WhitePublic.ManaMax
		m.WhitePublic.KingPlayed = false

		m.PassPrior = false

		m.WhitePrivate.Cards = drawCards(white, m.WhitePrivate.Cards)
		m.BlackPrivate.Cards = drawCards(black, m.BlackPrivate.Cards)

		m.RemoveNonPawns()
		// todo: spawn more pawns

	} else {
		if m.Turn == black {
			m.Turn = white
		} else {
			m.Turn = black
		}
		m.PassPrior = pass

		if m.WhitePublic.KingPlayed {
			m.WhitePrivate.PlayerInstruction = defaultInstruction
			m.WhitePrivate.SelectedCard = -1
			m.WhitePrivate.HighlightEmpty = false
		} else {
			m.WhitePrivate.PlayerInstruction = kingInstruction
			m.WhitePrivate.SelectedCard = 0
			m.WhitePrivate.HighlightEmpty = true
		}

		if m.BlackPublic.KingPlayed {
			m.BlackPrivate.PlayerInstruction = defaultInstruction
			m.BlackPrivate.SelectedCard = -1
			m.BlackPrivate.HighlightEmpty = false
		} else {
			m.BlackPrivate.PlayerInstruction = kingInstruction
			m.BlackPrivate.SelectedCard = 0
			m.BlackPrivate.HighlightEmpty = true
		}
	}
}

// return true if message triggers end of match
func processMessage(msg []byte, match *Match, player string) bool {
	var event string
	var notifyOpponent bool // set to true for events where opponent should get state update
	idx := 0
	for ; idx < len(msg); idx++ {
		if msg[idx] == ' ' {
			event = string(msg[:idx])
			msg = msg[idx+1:]
		}
	}
	if event == "" {
		event = "bad_event"
	}
	match.Mutex.Lock()
	var private *PrivateState
	var public *PublicState
	if player == black {
		private = &match.BlackPrivate
		public = &match.BlackPublic
	} else {
		private = &match.WhitePrivate
		public = &match.WhitePublic
	}
	switch event {
	case "get_state":
	case "click_card":
		if player != match.Turn {
			break // ignore if not the player's turn
		}
		if !public.KingPlayed {
			// cannot select other cards until king is played
			break
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
	case "click_board":
		if player != match.Turn {
			break // ignore if not the player's turn
		}
		type ClickBoardEvent struct {
			X int
			Y int
		}
		var event ClickBoardEvent
		err := json.Unmarshal(msg, &event)
		if err != nil {
			break // todo: send error response
		}
		// ignore if not card selected
		if private.SelectedCard == -1 {
			break
		}
		// ignore clicks on occupied spaces
		if match.getPiece(Pos{event.X, event.Y}) != nil {
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
		case king:
			p = Piece{king, player}
			public.KingPlayed = true
		case bishop:
			p = Piece{bishop, player}
		case knight:
			p = Piece{knight, player}
		case rook:
			p = Piece{rook, player}

		}
		match.setPiece(Pos{event.X, event.Y}, p)
		// remove card
		private.RemoveCard(private.SelectedCard)
		match.EndTurn(false, player)
		notifyOpponent = true
	case "pass":
		if player != match.Turn {
			break // ignore if not the player's turn
		}
		if !public.KingPlayed {
			break // cannot pass when king has not been played
		}
		match.EndTurn(true, player)
		notifyOpponent = true
	}
	processConnection := func(conn *websocket.Conn, color string, private *PrivateState) {
		if conn != nil {
			response := gin.H{
				"color":        color,
				"board":        match.Board,
				"private":      private,
				"turn":         match.Turn,
				"winner":       match.Winner,
				"round":        match.Round,
				"lastMoveTime": match.LastMoveTime,
				"blackPublic":  match.BlackPublic,
				"whitePublic":  match.WhitePublic,
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
		processConnection(match.BlackConn, black, &match.BlackPrivate)
		if notifyOpponent {
			processConnection(match.WhiteConn, white, &match.WhitePrivate)
		}
	} else {
		processConnection(match.WhiteConn, white, &match.WhitePrivate)
		if notifyOpponent {
			processConnection(match.BlackConn, black, &match.BlackPrivate)
		}
	}

	match.Mutex.Unlock()
	return false
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

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/static", "static")

	// list open matches
	router.GET("/", func(c *gin.Context) {
		type openMatch struct {
			Name string
			UUID string
		}
		liveMatches.Lock()
		openMatches := []openMatch{}
		for _, m := range liveMatches.internal {
			if m.IsOpen() {
				openMatches = append(openMatches, openMatch{m.Name, m.UUID})
			}
		}
		liveMatches.Unlock()

		c.HTML(http.StatusOK, "browse.tmpl", openMatches)
	})

	// browse non-open matches
	router.GET("/rest", func(c *gin.Context) {
		type openMatch struct {
			Name     string
			UUID     string
			Finished bool
		}
		liveMatches.Lock()
		openMatches := []openMatch{}
		for _, m := range liveMatches.internal {
			if m.IsFull() {
				openMatches = append(openMatches, openMatch{m.Name, m.UUID, false})
			}
			if m.IsFinished() {
				openMatches = append(openMatches, openMatch{m.Name, m.UUID, true})
			}
		}
		liveMatches.Unlock()

		c.HTML(http.StatusOK, "browse_rest.tmpl", openMatches)
	})

	// periodically clean liveMatches of finished or timedout games
	go func() {
		for {
			liveMatches.Lock()
			for id, match := range liveMatches.internal {
				if match.Winner != none ||
					time.Now().After(match.LastMoveTime.Add(matchTimeout)) {
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
		// todo give match a random name (adjective--animal)
		match := &Match{
			UUID: u4Str,
		}
		initMatch(match)
		liveMatches.Store(u4Str, match)
		c.Redirect(http.StatusSeeOther, "/match/"+u4Str)
	})

	// pass in UUID and optionally a password (from cookie? get param?)
	router.GET("/match/:id", func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("joining match: %v\n", id)
		if _, ok := liveMatches.Load(id); !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", id)
			return
		}
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	router.GET("/ws/:id", func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("making match connection: " + id)
		match, ok := liveMatches.Load(id)
		if !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", id)
			return
		}

		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Printf("Failed to set websocket upgrade: %+v", err)
			return
		}

		var player string

		match.Mutex.Lock()
		if match.BlackConn != nil && match.WhiteConn != nil {
			err := conn.WriteMessage(websocket.TextMessage, []byte("Match is full."))
			if err != nil {
				fmt.Printf("Error sending 'match full' message: %+v", err)
			}
			match.Mutex.Unlock()
			goto exit
		} else if match.BlackConn == nil && match.WhiteConn == nil {
			if rand.Intn(2) == 0 {
				match.BlackConn = conn
				player = black
			} else {
				match.WhiteConn = conn
				player = white
			}
		} else if match.BlackConn == nil {
			match.BlackConn = conn
			player = black
		} else if match.WhiteConn == nil {
			match.WhiteConn = conn
			player = white
		}
		match.Mutex.Unlock()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if processMessage(msg, match, player) {
				break
			}
		}

	exit:
		conn.Close()
		match.Mutex.Lock()
		if player == black {
			match.BlackConn = nil
		} else if player == white {
			match.WhiteConn = nil
		}
		match.Mutex.Unlock()
		fmt.Printf("Closed connection %s in match %s %s", player, match.Name, match.UUID)
	})

	router.Run(":" + port)
}
