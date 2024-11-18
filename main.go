package main

var (
	connections []*Connection = []*Connection{}
	games       []*Game       = []*Game{}
	rounds      []*GameRound  = []*GameRound{}
	players     []*Player     = []*Player{}
	answers     []*Answer     = []*Answer{}
)

type SocketMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Player struct {
	ID       string `bson:"id" json:"id"`
	Nickname string `bson:"nickname" json:"nickname"`
}

type Game struct {
	ID            string   `bson:"id" json:"id"`
	Name          string   `bson:"name" json:"name"`
	ModeratorUUID string   `bson:"moderatorId" json:"moderatorId"`
	Players       []string `bson:"players" json:"players"`
}

type GameRound struct {
	ID       string   `bson:"id" json:"id"`
	GameID   string   `bson:"gameId" json:"gameId"`
	Round    int      `bson:"round" json:"round"`
	Active   bool     `bson:"active" json:"active"`
	Question string   `bson:"question" json:"question"`
	Answers  []Answer `bson:"answers" json:"answers"`
	Started  bool     `bson:"started" json:"started"`
	Ended    bool     `bson:"ended" json:"ended"`
}

type Answer struct {
	ID                string `bson:"id" json:"id"`
	GameID            string `bson:"gameId" json:"gameId"`
	PlayerID          string `bson:"playerId" json:"playerId"`
	RoundID           string `bson:"roundId" json:"roundId"`
	Text              string `bson:"text" json:"text"`
	RevealedToPlayers bool   `bson:"revealedToPlayers" json:"revealedToPlayers"`
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
