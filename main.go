package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	connections  []*Connection = []*Connection{}
	game_texts   []GameText    = []GameText{}
	games        []Game        = []Game{}
	allowd_games []string      = []string{}
)

type AllAnswer struct {
	UUID   string `json:"uuid"`
	Nick   string `json:"nickname"`
	Answer string `json:"answer"`
}

type GameText struct {
	Text   string `json:"text"`
	GameId string `json:"gameId"`
}

type Answer struct {
	Answer string `json:"answer"`
	GameId string `json:"gameId"`
}

type SocketMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Player struct {
	UUID   string `json:"uuid"`
	Nick   string `json:"nickname"`
	GameId string `json:"gameId"`
}

type CreateGameRequest struct {
	Name          string `json:"name"`
	ModeratorUUID string `json:"uuid"`
}

type Game struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ModeratorUUID string `json:"uuid"`
}

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s %s %s %s", c.Request.Method, c.Request.URL, c.Request.Proto, c.Request.RemoteAddr)
		c.Next()
	}
}

func main() {
	upgrader := websocket.Upgrader{}
	router := gin.New()

	router.Use(Logging())
	router.Use(cors.New(cors.Config{
		AllowOrigins:    []string{"*"},
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowWebSockets: true,
		AllowFiles:      true,
	}))
	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		defer conn.Close()
		log.Println("connected")

		con := Connection{
			Conn:            conn,
			GameId:          "",
			UUID:            "",
			Nick:            "",
			IsModerator:     false,
			Answer:          "",
			AnswerRevielead: false,
		}

		connections = append(connections, &con)

		con.Listen()
	})
	router.GET("/game/:id", func(c *gin.Context) {
		id := c.Param("id")

		game, err := GetGame(id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, game)
	})
	router.POST("/game", func(c *gin.Context) {
		var req CreateGameRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id := CreateGame(req.Name, req.ModeratorUUID)

		c.JSON(http.StatusOK, gin.H{
			"id": id,
		})
	})
	router.Static("/assets", "./public/assets")
	router.StaticFile("/", "./public/index.html")
	router.StaticFile("/vite.svg", "./public/vite.svg")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %s", err)
	}
}
