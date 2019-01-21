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
	Name             string // used to identify the match in browser
	BlackConn        *websocket.Conn
	WhiteConn        *websocket.Conn
	Mutex            sync.RWMutex
	Pieces           []Piece
	CommunalCards    []Card // card in pool shared by both players
	StartTime        time.Time
	UUID             string
	BlackState       PrivateState
	WhiteState       PrivateState
	Turn             string    `json:"turn"`         // white, black
	Winner           string    `json:"winner"`       // white, black, none, draw
	LastMoveTime     time.Time `json:"lastMoveTime"` // should be initialized to match start time
	WhiteManaMax     int       `json:"whiteManaMax"`
	BlackManaMax     int       `json:"blackManaMax"`
	WhiteManaCurrent int       `json:"whiteManaCurrent"`
	BlackManaCurrent int       `json:"blackManaCurrent"`
}

const (
	white = "white"
	black = "black"
	draw  = "draw"
	none  = "none"
)

const (
	pawn   = "pawn"
	king   = "king"
	queen  = "queen"
	rook   = "rook"
	bishop = "bishop"
	knight = "knight"
)

const nColumns = 6
const nRow = 6

// info a player doesn't want opponent to see
type PrivateState struct {
	Cards        []Card `json:"cards"`
	SelectedCard int    `json:"selectedCard"` // index into cards slice
	SelectedPos  Pos    `json:"selectedPos"`
	AttackPos    []Pos  `json:"attackPos"`
}

type Piece struct {
	Type       string `json:"type"`
	Color      string `json:"color"`
	Pos        Pos    `json:"pos"`
	ValidMoves []Pos  `json:"validMoves"`
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

func drawCards(owner string) []Card {
	return []Card{
		Card{"King", owner, 0},
		Card{"Bishop", owner, 0},
		Card{"Knight", owner, 0},
		Card{"Rook", owner, 0},
	}
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
	pieces := make([]Piece, 8)
	for i := 0; i < 4; i++ {
		pieces[i] = Piece{pawn, white, Pos{columns[i], rand.Intn(2) + 1}, nil}
	}
	columns = make([]int, nColumns)
	for i := 0; i < nColumns; i++ {
		columns[i] = i
	}
	columns = randSelect(4, columns)
	for i := 0; i < 4; i++ {
		pieces[i+4] = Piece{pawn, black, Pos{columns[i], rand.Intn(2) + 3}, nil}
	}
	m.Pieces = pieces

	m.WhiteManaCurrent = 3
	m.WhiteManaMax = 3
	m.BlackManaCurrent = 3
	m.BlackManaMax = 3

	m.CommunalCards = drawCards(none)
	m.BlackState = PrivateState{
		Cards:        drawCards(black),
		SelectedCard: -1,
		SelectedPos:  Pos{-1, -1},
	}
	m.WhiteState = PrivateState{
		Cards:        drawCards(white),
		SelectedCard: -1,
		SelectedPos:  Pos{-1, -1},
	}
	m.StartTime = time.Now()
	m.Turn = white
	m.Winner = none
	m.LastMoveTime = m.StartTime
}

// return true if message triggers end of match
func processMessage(msg []byte, match *Match, player string) bool {
	var event string
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
	if player == black {
		private = &match.BlackState
	} else {
		private = &match.WhiteState
	}
	switch event {
	case "get_state":
	case "click_card":
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
		} else {
			private.SelectedCard = event.SelectedCard
		}
	case "click_board":
		type ClickBoardEvent struct {
			SelectedCard int
		}
		var event ClickBoardEvent
		err := json.Unmarshal(msg, &event)
		if err != nil {
			break // todo: send error response
		}
		// todo
	}
	processConnection := func(conn *websocket.Conn, color string, private *PrivateState) {
		if conn != nil {
			response := gin.H{
				"color":            color,
				"pieces":           match.Pieces,
				"private":          private,
				"turn":             match.Turn,
				"winner":           match.Winner,
				"lastMoveTime":     match.LastMoveTime,
				"blackManaCurrent": match.BlackManaCurrent,
				"blackManaMax":     match.BlackManaMax,
				"whiteManaCurrent": match.WhiteManaCurrent,
				"whiteManaMax":     match.WhiteManaMax,
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
	processConnection(match.BlackConn, black, &match.BlackState)
	processConnection(match.WhiteConn, white, &match.WhiteState)
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
