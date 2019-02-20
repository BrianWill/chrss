package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"

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
				"boardStatus":               match.SquareStatuses,
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

func (mm *MatchMap) Load(key string) (*Match, bool) {
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

func NewUserMap() *UserMap {
	return &UserMap{
		internal: make(map[string]bool),
	}
}

func (um *UserMap) Exists(key string) bool {
	um.RLock()
	_, ok := um.internal[key]
	um.RUnlock()
	return ok
}

func (um *UserMap) Delete(key string) {
	um.Lock()
	delete(um.internal, key)
	um.Unlock()
}

func (um *UserMap) Store(userID string) {
	um.Lock()
	um.internal[userID] = true
	um.Unlock()
}

var userNumber = 1 // used for default user names
// returns userID (which may be new if argument does not exist
func validateUser(c *gin.Context, userID string, userName string, users *UserMap) (string, string, error) {
	users.Lock()
	if !users.internal[userID] {
		u2, err := uuid.NewV4()
		if err != nil {
			users.Unlock()
			return "", "", err
		}
		userID = u2.String()
		const tenYears = 10 * 365 * 24 * 60 * 60
		c.SetCookie("user_id", userID, tenYears, "/", "", false, false)
		userName = strconv.Itoa(userNumber)
		c.SetCookie("user_name", strconv.Itoa(userNumber), tenYears, "/", "", false, false)
		userNumber++
	}
	users.internal[userID] = true
	users.Unlock()
	return userID, userName, nil
}

func createMatch(c *gin.Context, liveMatches *MatchMap, users *UserMap) (string, error) {
	userID, err := c.Cookie("user_id")
	userName, _ := c.Cookie("user_name")
	userID, userName, err = validateUser(c, userID, userName, users)
	if err != nil {
		return "", err
	}

	name := adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]

	liveMatches.Lock()
	// if name collision with existing match, randomly generate new names until finding one that's not in use
	// (not ideal, but this is partly why we limit number of active matches)
	for _, ok := liveMatches.internal[name]; ok; {
		name = adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
	}
	match := &Match{
		Name:          name,
		WhitePlayerID: userID,
		BlackPlayerID: userID,
		CreatorName:   userName,
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
		return "", errors.New("At max matches. Cannot create an additional match.")
	}

	initMatch(match, false)
	liveMatches.Store(match)
	return match.Name, nil
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
	users := NewUserMap()

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		userID, err := c.Cookie("user_id")
		userName, _ := c.Cookie("user_name")
		userID, userName, err = validateUser(c, userID, userName, users)
		if err != nil {
			fmt.Printf("Error generating UUIDv4: %s", err)
			return
		}

		fmt.Printf("User id: %s User name: %s \n", userID, userName)

		now := time.Now()
		type match struct {
			Name        string
			CreatorName string
			StartTime   int64
			Elapsed     string
			Color       string
		}
		liveMatches.Lock()
		matches := []match{}
		playerMatches := []match{}
		for _, m := range liveMatches.internal {
			elapsed := fmtDuration(now.Sub(time.Unix(0, m.StartTime)))
			if m.IsBlackOpen() {
				matches = append(matches, match{m.Name, m.CreatorName, m.StartTime, elapsed, none})
			}
			if m.BlackPlayerID == userID {
				playerMatches = append(playerMatches, match{m.Name, m.CreatorName, m.StartTime, elapsed, black})
			} else if m.WhitePlayerID == userID {
				playerMatches = append(playerMatches, match{m.Name, m.CreatorName, m.StartTime, elapsed, white})
			}
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].StartTime > matches[j].StartTime })
		liveMatches.Unlock()

		c.HTML(http.StatusOK, "home.tmpl", struct {
			ID            string
			Name          string
			Matches       []match
			PlayerMatches []match
		}{userID, userName, matches, playerMatches})
	})

	router.GET("/guide", func(c *gin.Context) {
		c.HTML(http.StatusOK, "guide.tmpl", nil)
	})

	router.GET("/createMatch", func(c *gin.Context) {
		name, err := createMatch(c, liveMatches, users)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		c.Redirect(http.StatusSeeOther, "/match/"+name+"/white")
	})

	router.GET("/dev", func(c *gin.Context) {
		name, err := createMatch(c, liveMatches, users)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		c.Redirect(http.StatusSeeOther, "/dev/"+name)
	})

	router.GET("/dev/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.HTML(http.StatusOK, "dev.tmpl", name)
	})

	router.GET("/match/:name/:color", func(c *gin.Context) {
		userID, err := c.Cookie("user_id")
		userName, _ := c.Cookie("user_name")
		userID, userName, err = validateUser(c, userID, userName, users)
		if err != nil {
			fmt.Printf("Error generating UUIDv4: %s", err)
			return
		}

		name := c.Param("name")
		color := c.Param("color")
		if color != "black" && color != "white" {
			c.String(http.StatusNotFound, "Must specify black or white. Invalid match color: '%s'.", color)
			return
		}
		log.Printf("joining match: %v\n", name)

		match, ok := liveMatches.Load(name)
		match.Mutex.Lock()
		if !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", name)
			match.Mutex.Unlock()
			return
		}
		if color == "black" {
			if match.BlackPlayerID == "" {
				match.BlackPlayerID = userID
			} else if match.BlackPlayerID != userID {
				c.String(http.StatusBadRequest,
					"Cannot join match '%s' as black. Another player is already playing that color.", name,
				)
				match.Mutex.Unlock()
				return
			}
		} else {
			if match.WhitePlayerID == "" {
				match.WhitePlayerID = userID
			} else if match.WhitePlayerID != userID {
				c.String(http.StatusBadRequest,
					"Cannot join match '%s' as white. Another player is already playing that color.", name,
				)
				match.Mutex.Unlock()
				return
			}
		}
		match.Mutex.Unlock()
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	router.GET("/ws/:name/:color", func(c *gin.Context) {
		userID, err := c.Cookie("user_id")
		userName, _ := c.Cookie("user_name")
		userID, userName, err = validateUser(c, userID, userName, users)
		if err != nil {
			fmt.Printf("Error generating UUIDv4: %s", err)
			return
		}

		name := c.Param("name")
		color := c.Param("color")
		if color != "black" && color != "white" {
			c.String(http.StatusNotFound, "Must specify black or white. Invalid match color: '%s'.", color)
			return
		}
		log.Printf("joining match: %v\n", name)

		match, ok := liveMatches.Load(name)
		if !ok {
			c.String(http.StatusNotFound, "No match with id '%s' exists.", name)
			return
		}
		match.Mutex.Lock()
		if color == "black" {
			if match.BlackPlayerID != userID {
				c.String(http.StatusBadRequest,
					"Cannot join match '%s' as black. Another player is already playing that color.", name,
				)
				match.Mutex.Unlock()
				return
			}
		} else {
			if match.WhitePlayerID != userID {
				c.String(http.StatusBadRequest,
					"Cannot join match '%s' as white. Another player is already playing that color.", name,
				)
				match.Mutex.Unlock()
				return
			}
		}

		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Printf("Failed to set websocket upgrade: %+v", err)
			match.Mutex.Unlock()
			return
		}
		// if client is valid, we kill previous websocket to start new one
		if color == black {
			if match.BlackConn != nil {
				match.BlackConn.Close()
				fmt.Printf("Closed black connection in match '%s' ", match.Name)
			}
			match.BlackConn = conn
		} else {
			if match.WhiteConn != nil {
				match.WhiteConn.Close()
				fmt.Printf("Closed white connection in match '%s' ", match.Name)
			}
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

		match.Mutex.Lock()
		conn.Close()
		if color == black {
			// a subsequent request may have replaced this conn, so we check
			if match.BlackConn == conn {
				match.BlackConn = nil
			}
		} else if color == white {
			if match.WhiteConn == conn {
				match.WhiteConn = nil
			}
		}
		fmt.Printf("Closed connection '%s' in match %s ", color, match.Name)
		match.Mutex.Unlock()
	})

	router.Run(":" + port)
}
