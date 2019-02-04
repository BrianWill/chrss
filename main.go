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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/heroku/x/hmetrics/onload"
)

var positions [nColumns * nRows]Pos // convenience for getting Pos of board index

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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

func processMessage(msg []byte, match *Match, player string) {
	currentRound := match.Round
	var event string
	idx := 0
	for ; idx < len(msg); idx++ {
		if msg[idx] == ' ' {
			event = string(msg[:idx])
			msg = msg[idx+1:]
		}
	}
	if event == "ping" {
		// used for keep alive (heroku timesout connections with no activity for 55 seconds)
		// needn't send response to keep connection alive
		return
	}
	match.Mutex.Lock()
	notifyOpponent, newTurn := match.processEvent(event, player, msg)
	processConnection := func(conn *websocket.Conn, color string, private *PrivateState, newTurn bool) {
		turnElapsed := time.Now().UnixNano() - match.LastMoveTime
		remainingTurnTime := (turnTimer - turnElapsed) / 1000000
		if conn != nil {
			response := gin.H{
				"turnRemainingMilliseconds": remainingTurnTime,
				"color":                     color,
				"board":                     match.Board,
				"boardStatus":               match.Combined,
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

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	s := ""
	if h > 0 {
		s += strconv.Itoa(int(h)) + " hrs "
	}
	s += strconv.Itoa(int(m)) + " min"
	return s
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

func (mm *MatchMap) Store(match *Match) {
	mm.Lock()
	mm.internal[match.Name] = match
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
		now := time.Now()
		type match struct {
			Name      string
			BlackOpen bool
			WhiteOpen bool
			StartTime int64
			Elapsed   string
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
			elapsed := fmtDuration(now.Sub(time.Unix(0, m.StartTime)))
			if black || white {
				matches.Open = append(matches.Open, match{m.Name, black, white, m.StartTime, elapsed})
			} else {
				matches.Full = append(matches.Full, match{m.Name, black, white, m.StartTime, elapsed})
			}
			i++
		}
		sort.Slice(matches.Open, func(i, j int) bool { return matches.Open[i].StartTime > matches.Open[j].StartTime })
		sort.Slice(matches.Full, func(i, j int) bool { return matches.Full[i].StartTime > matches.Full[j].StartTime })
		liveMatches.Unlock()

		c.HTML(http.StatusOK, "browse.tmpl", matches)
	})

	router.GET("/guide", func(c *gin.Context) {
		c.HTML(http.StatusOK, "guide.tmpl", nil)
	})

	router.GET("/createMatch", func(c *gin.Context) {
		name := adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
		liveMatches.Lock()
		// if name collision with existing match, randomly generate new names until finding one that's not in use
		// (not ideal, but this is partly why we limit number of active matches)
		for _, ok := liveMatches.internal[name]; ok; {
			name = adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
		}
		match := &Match{
			Name: name,
		}
		// clean up any dead or timedout matches
		for name, match := range liveMatches.internal {
			exceededTimeout := time.Now().UnixNano() > match.LastMoveTime+matchTimeout
			if match.Phase == gameoverPhase || exceededTimeout {
				liveMatches.internal[name].Mutex.Lock()
				delete(liveMatches.internal, name)
			}
		}
		nMatches := len(liveMatches.internal)
		liveMatches.Unlock()

		if nMatches >= maxConcurrentMatches {
			c.String(http.StatusInternalServerError, "Cannot create match. Server currently at max number of matches.")
			return
		}

		// new match
		initMatch(match)
		liveMatches.Store(match)

		c.Redirect(http.StatusSeeOther, "/")
	})

	router.GET("/dev/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.HTML(http.StatusOK, "dev.tmpl", name)
	})

	// pass in UUID and optionally a password (from cookie? get param?)
	router.GET("/match/:name/:color", func(c *gin.Context) {
		name := c.Param("name")
		color := c.Param("color")
		if color != "black" && color != "white" {
			c.String(http.StatusNotFound, "Must specify black or white. Invalid match color: '%s'.", color)
			return
		}
		log.Printf("joining match: %v\n", name)
		if _, ok := liveMatches.Load(name); !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", name)
			return
		}
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	router.GET("/ws/:name/:color", func(c *gin.Context) {
		name := c.Param("name")
		color := c.Param("color")
		log.Printf("making match connection: " + name + " " + color)
		if color != "black" && color != "white" {
			c.String(http.StatusNotFound, "Must specify black or white. Invalid match color: '%s'.", color)
			return
		}
		match, ok := liveMatches.Load(name)
		if !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", name)
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
		fmt.Printf("Closed connection '%s' in match %s ", color, match.Name)
		match.Mutex.Unlock()
	})

	router.Run(":" + port)
}
