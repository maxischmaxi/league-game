package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	idParam := c.Param("id")

	if idParam == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	id, err := primitive.ObjectIDFromHex(idParam)

	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	coll := s.Database.Database("league").Collection("games")
	filter := bson.D{{Key: "_id", Value: id}}
	var result Game

	err = coll.FindOne(c, filter).Decode(&result)

	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, result)
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
		Conn:     conn,
		PlayerID: nil,
	}

	connections = append(connections, &con)

	con.Listen(s.Database)
}
