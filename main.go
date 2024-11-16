package main

import (
	"log"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	connections []*Connection = []*Connection{}
)

type AllAnswer struct {
	UUID           string `json:"uuid"`
	Nick           string `json:"nickname"`
	Answer         string `json:"answer"`
	AnswerRevealed bool   `json:"answerRevealed"`
}

type SetPreviewdPayload struct {
	GameId  string `json:"gameId"`
	Preview bool   `json:"preview"`
}

type SocketMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Player struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Nickname string             `json:"nickname"`
}

type Game struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name          string               `bson:"name" json:"name"`
	ModeratorUUID primitive.ObjectID   `bson:"moderatorId" json:"moderatorId"`
	Players       []primitive.ObjectID `bson:"players" json:"players"`
}

type GameRound struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GameID   primitive.ObjectID `bson:"gameId" json:"gameId"`
	Round    int                `bson:"round" json:"round"`
	Active   bool               `bson:"active" json:"active"`
	Question string             `bson:"question" json:"question"`
	Answers  []Answer           `bson:"answers" json:"answers"`
	Started  bool               `bson:"started" json:"started"`
	Ended    bool               `bson:"ended" json:"ended"`
}

type Answer struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GameID            primitive.ObjectID `bson:"gameId" json:"gameId"`
	PlayerID          primitive.ObjectID `bson:"playerId" json:"playerId"`
	RoundID           primitive.ObjectID `bson:"roundId" json:"roundId"`
	Text              string             `bson:"text" json:"text"`
	RevealedToPlayers bool               `bson:"revealedToPlayers" json:"revealedToPlayers"`
}

func main() {
	client, err := InitDatabase()

	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}

	defer DisconnectDatabase(client)

	router := NewServer(client)

	router.GET("/ws", router.HandleWebsocket)
	router.GET("/game/:id", router.GetGameById)
	router.Static("/assets", "./public/assets")
	router.StaticFile("/", "./public/index.html")
	router.StaticFile("/vite.svg", "./public/vite.svg")

	router.RunWithLogs()
}
