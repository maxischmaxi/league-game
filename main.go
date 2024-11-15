package main

import (
	"log"
)

var (
	connections   []*Connection = []*Connection{}
	game_texts    []GameText    = []GameText{}
	games         []Game        = []Game{}
	allowed_games []string      = []string{}
	previewed     []string      = []string{}
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

func main() {
	client, err := InitDatabase()

	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}

	defer DisconnectDatabase(client)

	router := NewServer(client)

	router.GET("/ws", router.HandleWebsocket)
	router.GET("/game/:id", router.GetGameById)
	router.GET("/reset", router.Reset)
	router.POST("/game", router.CreateGame)
	router.Static("/assets", "./public/assets")
	router.StaticFile("/", "./public/index.html")
	router.StaticFile("/vite.svg", "./public/vite.svg")

	router.RunWithLogs()
}
