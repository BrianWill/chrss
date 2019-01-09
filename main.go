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

const matchTimeout time.Duration = 20 * time.Minute

type Match struct {
	Name       string // used to identify the match in browser
	BlackConn  *websocket.Conn
	WhiteConn  *websocket.Conn
	Mutex      sync.RWMutex
	Pieces     []Piece
	Cards      []Card
	State      MatchState
	UUID       string
	BlackState PrivateState
	WhiteState PrivateState
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

// info a player doesn't want opponent to see
type PrivateState struct {
}

type Piece struct {
	Type  string `json:"type"`
	Color string `json:"color"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type Card struct {
	Name     string `json:"name"`
	Owner    string `json:"owner"` // black, white, none
	ManaCost int    `json:"manaCost"`
}

type MatchState struct {
	Turn     string    `json:"turn"`     // white, black
	Winner   string    `json:"winner"`   // white, black, none, draw
	LastMove time.Time `json:"lastMove"` // should be initialized to match start time
}

type MatchMap struct {
	sync.RWMutex
	internal map[string]*Match
}

func drawCards() []Card {
	return nil
}

func initMatch(m *Match) {
	// random adjective-animal
	m.Name = adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
	m.State.LastMove = time.Now()
	m.Pieces = []Piece{
		Piece{king, white, 4, 0},
		Piece{rook, white, 7, 0},
		Piece{knight, white, 6, 0},
		Piece{bishop, white, 5, 0},
		Piece{king, black, 3, 7},
		Piece{rook, black, 0, 7},
		Piece{knight, black, 1, 7},
		Piece{bishop, black, 2, 7},
	}
	m.Cards = drawCards()
	m.State = MatchState{
		Turn:     white,
		Winner:   none,
		LastMove: time.Now(),
	}
}

// return true if message triggers end of match
func processMessage(msg []byte, match *Match) bool {
	match.Mutex.Lock()

	if match.BlackConn != nil {
		response := gin.H{
			"color":   "black",
			"private": match.BlackState,
			"pieces":  match.Pieces,
			"cards":   match.Cards,
			"state":   match.State,
		}
		bytes, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("Error JSON encoding state: %+v", err)
		}
		err = match.BlackConn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			fmt.Printf("Error writing message to black connection: %+v", err)
		}
	}
	if match.WhiteConn != nil {
		response := gin.H{
			"color":   "white",
			"private": match.WhiteState,
			"pieces":  match.Pieces,
			"cards":   match.Cards,
			"state":   match.State,
		}
		bytes, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("Error JSON encoding state: %+v", err)
		}
		err = match.WhiteConn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			fmt.Printf("Error writing message to white connection: %+v", err)
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

	fmt.Printf("router type %T\n", router)

	router.GET("/", func(c *gin.Context) {
		// todo show matches
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	// periodically clean liveMatches of finished or timedout games
	go func() {
		for {
			liveMatches.Lock()
			for id, match := range liveMatches.internal {
				if match.State.Winner != none ||
					time.Now().After(match.State.LastMove.Add(matchTimeout)) {
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
		log.Printf("joining match: " + id)
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
			} else {
				match.WhiteConn = conn
			}
		} else if match.BlackConn == nil {
			match.BlackConn = conn
		} else if match.WhiteConn == nil {
			match.WhiteConn = conn
		}
		match.Mutex.Unlock()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if processMessage(msg, match) {
				break
			}
		}

	exit:
		conn.Close()
	})

	router.Run(":" + port)
}
