package main

import ()

var (
	connections []*Connection = []*Connection{}
	games       []*Game       = []*Game{}
	rounds      []*GameRound  = []*GameRound{}
	players     []*Player     = []*Player{}
	answers     []*Answer     = []*Answer{}
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
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type Game struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ModeratorUUID string   `json:"moderatorId"`
	Players       []string `json:"players"`
}

type GameRound struct {
	ID       string   `json:"id"`
	GameID   string   `json:"gameId"`
	Round    int      `json:"round"`
	Active   bool     `json:"active"`
	Question string   `json:"question"`
	Answers  []Answer `json:"answers"`
	Started  bool     `json:"started"`
	Ended    bool     `json:"ended"`
}

type Answer struct {
	ID                string `json:"id"`
	GameID            string `json:"gameId"`
	PlayerID          string `json:"playerId"`
	RoundID           string `json:"roundId"`
	Text              string `json:"text"`
	RevealedToPlayers bool   `json:"revealedToPlayers"`
}

func main() {
	router := NewServer()

	router.GET("/ws", router.HandleWebsocket)
	router.GET("/game/:id", router.GetGameById)
	router.Static("/assets", "./public/assets")
	router.StaticFile("/", "./public/index.html")
	router.StaticFile("/vite.svg", "./public/vite.svg")

	router.RunWithLogs()
}
