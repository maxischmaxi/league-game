package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
)

type Server struct {
	*gin.Engine
	Upgrader websocket.Upgrader
	Database *mongo.Client
}

func NewServer(client *mongo.Client) *Server {
	server := gin.New()

	server.Use(cors.New(cors.Config{
		AllowOrigins:    []string{"https://league-game.up.railway.app", "http://localhost:5173"},
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "X-Requested-With", "X-CSRF-Token", "Authorization", "Token", "Host", "Connection", "Accept-Encoding", "Accept-Language", "DNT", "Sec-Fetch-Mode", "Sec-Fetch-Site", "Sec-Fetch-Dest", "Referer", "User-Agent"},
		AllowWebSockets: true,
		AllowFiles:      true,
		MaxAge:          12 * time.Hour,
	}))

	return &Server{
		server,
		websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")

				allowed := []string{"https://league-game.up.railway.app", "http://localhost:5173"}

				for _, v := range allowed {
					if v == origin {
						return true
					}
				}

				return false
			},
		},
		client,
	}
}

func (s *Server) GetGameById(c *gin.Context) {
	id := c.Param("id")

	game, err := GetGame(id)

	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, game)
}

func (s *Server) Reset(c *gin.Context) {
	for _, conn := range connections {
		err := conn.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

		if err != nil {
			log.Println("write:", err)
		}
	}

	connections = []*Connection{}

	games = []Game{}
	game_texts = []GameText{}
	allowed_games = []string{}
}

func (s *Server) CreateGame(c *gin.Context) {
	var req CreateGameRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := CreateGame(req.Name, req.ModeratorUUID)

	c.JSON(http.StatusOK, gin.H{
		"id": id,
	})
}

func (s *Server) RunWithLogs() {
	if err := s.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %s", err)
	}
}

func (s *Server) HandleWebsocket(c *gin.Context) {
	conn, err := s.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		err := c.AbortWithError(http.StatusInternalServerError, err)

		if err != nil {
			log.Println("upgrade:", err)
		}

		return
	}

	defer conn.Close()
	log.Println("connected")

	con := Connection{
		Conn:           conn,
		GameId:         "",
		UUID:           "",
		Nick:           "",
		IsModerator:    false,
		Answer:         "",
		AnswerRevealed: false,
	}

	connections = append(connections, &con)

	con.Listen(s.Database)
}
