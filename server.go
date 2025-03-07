package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
	*gin.Engine
	Upgrader websocket.Upgrader
}

func NewServer() *Server {
	server := gin.New()

	server.Use(gzip.Gzip(gzip.DefaultCompression))

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
	}
}

func (s *Server) GetGameById(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	for _, g := range games {
		if g.ID == id {
			c.JSON(http.StatusOK, g)
			return
		}
	}

	c.AbortWithStatus(http.StatusNotFound)
}

func (s *Server) RunWithLogs() {
	if err := s.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %s", err)
	}
}

func (s *Server) HandleWebsocket(c *gin.Context) {
	cookie, err := c.Request.Cookie("uuid")
	var playerId *string

	if err != nil {
		fmt.Printf("Failed to get cookie: %s\n", err)
	} else {
		playerId = &cookie.Value
	}

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
		Conn:     conn,
		PlayerID: playerId,
	}

	connections = append(connections, &con)

	con.Listen()
}

type HelloBody struct {
	Nickname string `json:"nickname"`
	UUID     string `json:"uuid"`
}

type HelloResponse struct {
	UUID             string      `json:"uuid"`
	Nickname         string      `json:"nickname"`
	Games            []Game      `json:"games"`
	ConnectedPlayers []Player    `json:"connectedPlayers"`
	Rounds           []GameRound `json:"rounds"`
	Round            GameRound   `json:"round"`
	Text             string      `json:"text"`
	Game             Game        `json:"game"`
	Answers          []Answer    `json:"answers"`
}
