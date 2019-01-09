package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	liveMatches := NewMatchMap()

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	fmt.Printf("router type %T\n", router)

	router.GET("/", func(c *gin.Context) {
		// todo show matches
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	// periodically clean liveMatches of finished or timedout games
	go func() {
		for {
			liveMatches.Op(func(mm map[string]*Match) {
				for id, match := range mm {
					if match.State.Winner != none ||
						time.Now().After(match.State.LastMove.Add(matchTimeout)) {
						delete(mm, id)
					}
				}
			})
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
		match.State.LastMove = time.Now()
		liveMatches.Store(u4Str, match)
		c.Redirect(http.StatusSeeOther, "/match/"+u4Str)
		// c.Request.URL.Path = "/match/" + u4Str
		// router.HandleContext(c)
	})

	// pass in UUID and optionally a password (from cookie? get param?)
	router.GET("/match/:id", func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("joining match: " + id)
		if match, ok := liveMatches.Load(id); ok {
			if match.ClientBlack != nil && match.ClientWhite != nil {
				c.String(http.StatusForbidden, "That match is full.", nil)
			} else {
				c.HTML(http.StatusOK, "index.tmpl.html", nil)
			}
		} else {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", id)
		}
	})

	router.GET("/ws", func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Println("Failed to set websocket upgrade: %+v", err)
			return
		}

		for {
			t, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			msg, err = json.Marshal(true)
			if err != nil {
				fmt.Println("Failed to marshal match: %+v", err)
			}
			conn.WriteMessage(t, msg)
		}
	})

	router.Run(":" + port)
}

type Match struct {
	Name        string          `json:"name"` // used to identify the match in browser
	ClientBlack *websocket.Conn `json:"-"`
	ClientWhite *websocket.Conn `json:"-"`
	Pieces      []Piece         `json:"pieces"`
	Cards       []Card          `json:"cards"`
	State       MatchState      `json:"state"`
	UUID        string          `json:"uuid"`
}

const (
	white = "white"
	black = "black"
	draw  = "draw"
	none  = "none"
)

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

func (mm *MatchMap) Op(f func(map[string]*Match)) {
	mm.Lock()
	f(mm.internal)
	mm.Unlock()
}
