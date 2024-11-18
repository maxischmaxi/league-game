package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Connection struct {
	Conn     *websocket.Conn
	PlayerID *string
}

func (c *Connection) GetPlayer() (*Player, error) {
	if c.PlayerID == nil {
		return nil, fmt.Errorf("player id is nil")
	}

	for _, p := range players {
		if p.ID == *c.PlayerID {
			return p, nil
		}
	}

	return nil, fmt.Errorf("player not found")
}

func (c *Connection) Remove() {
	player, err := c.GetPlayer()

	if err != nil {
		log.Println("get player:", err)
		return
	}

	p, err := json.Marshal(player)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	msg := SocketMessage{
		Type:    "player_disconnected",
		Payload: string(p),
	}

	data, err := json.Marshal(msg)

	if err != nil {
		log.Println("marshal:", err)
	}

	for i, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		if conn.PlayerID == c.PlayerID {
			connections = append(connections[:i], connections[i+1:]...)
			continue
		}

		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			log.Println("write:", err)
		}
	}
}

func (c *Connection) SendJoinSuccess(game Game) error {
	data, err := json.Marshal(game)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer := SocketMessage{
		Type:    "join_game",
		Payload: string(data),
	}

	data, err = json.Marshal(answer)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))
	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

func (c *Connection) SendPlayerConnectedToAll(game Game, player Player) error {
	data, err := json.Marshal(player)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	res := SocketMessage{
		Type:    "player_connected",
		Payload: string(data),
	}

	data, err = json.Marshal(res)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		player, err := conn.GetPlayer()
		if err != nil {
			log.Println("get player:", err)
			continue
		}

		found := false

		for _, p := range game.Players {
			if p != player.ID {
				continue
			}

			found = true
		}

		if !found {
			continue
		}

		err = conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))
		if err != nil {
			log.Println("write:", err)
		}
	}

	return nil
}

func (c *Connection) HandleGameNotFound() error {
	answer := SocketMessage{
		Type:    "join_game",
		Payload: "false",
	}

	data, err := json.Marshal(answer)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))
	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

func (c *Connection) JoinGame(msg SocketMessage) {
	player, err := c.GetPlayer()
	if err != nil {
		log.Println("get player:", err)
		return
	}

	game, err := FindGameById(msg.Payload)

	if err != nil {
		log.Println("find one:", err)
		return
	}

	players := append(game.Players, *c.PlayerID)

	err = UpdateGamePlayers(game.ID, players)
	if err != nil {
		log.Println("update game players:", err)
		return
	}

	err = c.SendJoinSuccess(*game)
	if err != nil {
		log.Println("send join success:", err)
		return
	}

	err = c.SendPlayerConnectedToAll(*game, *player)
	if err != nil {
		log.Println("send player connected to all:", err)
		return
	}

	c.SendAllAnswers()
}

func (c *Connection) SendAllAnswers() {
	game, err := c.GetActiveGame()
	if err != nil {
		log.Println("get active game:", err)
		return
	}

	round, err := c.GetActiveRound()
	if err != nil {
		log.Println("get active round:", err)
		return
	}

	answers, err := FindAllAnswersByGameAndRound(game.ID, round.ID)
	if err != nil {
		log.Println("find all answers by game and round:", err)
		return
	}

	data, err := json.Marshal(answers)
	if err != nil {
		log.Println("marshal:", err)
		return
	}

	response := SocketMessage{
		Type:    "all_answers",
		Payload: string(data),
	}

	responseData, err := json.Marshal(response)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (c *Connection) UnhandledMessage(msg SocketMessage) {
	answer := SocketMessage{
		Type:    "error",
		Payload: "unknown message type",
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println("write:", err)
	}
}

func (c *Connection) GetActiveRound() (*GameRound, error) {
	game, err := c.GetActiveGame()

	if err != nil {
		log.Println("get active game:", err)
		return nil, err
	}

	round, err := FindActiveRoundByGameId(game.ID)

	if err != nil {
		log.Println("find one:", err)
		return nil, err
	}

	return round, nil
}

func (c *Connection) GetActiveGame() (*Game, error) {
	player, err := c.GetPlayer()

	if err != nil {
		log.Println("GetActiveGame: get player:", err)
		return nil, err
	}

	playerGames := []Game{}
	for _, g := range games {
		for _, p := range g.Players {
			if p == player.ID {
				playerGames = append(playerGames, *g)
			}
		}
	}

	if len(playerGames) == 0 {
		moderatorGames := []Game{}
		for _, g := range games {
			if g.ModeratorUUID == player.ID {
				moderatorGames = append(moderatorGames, *g)
			}
		}

		if len(moderatorGames) == 0 {
			return nil, fmt.Errorf("no active games")
		}

		if len(moderatorGames) > 1 {
			return nil, fmt.Errorf("multiple active games")
		}

		return &moderatorGames[0], nil
	}

	if len(playerGames) > 1 {
		return nil, fmt.Errorf("multiple active games")
	}

	return &playerGames[0], nil
}

func (c *Connection) SendConnectedPlayers() {
	game, err := c.GetActiveGame()
	if err != nil {
		log.Println("SEND_CONNECTED_PLAYERS: get active game:", err)
		return
	}

	playerIds := []string{}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		g, err := conn.GetActiveGame()

		if err != nil {
			log.Println("get active game:", err)
			continue
		}

		if game.ID != g.ID {
			continue
		}

		playerIds = append(playerIds, *conn.PlayerID)
	}

	selectedPlayers := []Player{}

	for _, p := range players {
		for _, id := range playerIds {
			if p.ID == id {
				selectedPlayers = append(selectedPlayers, *p)
			}
		}
	}

	payload, err := json.Marshal(selectedPlayers)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	answer := SocketMessage{
		Type:    "get_connected_players",
		Payload: string(payload),
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println("write:", err)
	}
}

type SayHelloPayload struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

func (c *Connection) SayHello(msg SocketMessage) {
	var payload SayHelloPayload
	err := json.Unmarshal([]byte(msg.Payload), &payload)
	if err != nil {
		log.Println("unmarshal:", err)
		return
	}

	fmt.Println("Player connected:", payload)

	if payload.UUID == "" {
		fmt.Println("player uuid is empty", payload.UUID)

		newPlayer := Player{
			Nickname: payload.Name,
			ID:       uuid.New().String(),
		}

		players = append(players, &newPlayer)
		c.PlayerID = &newPlayer.ID

		c.SendPlayerConnected(newPlayer)
		c.SendSetUuid()
		c.SendCurrentGame()
		c.SendAllAnswers()
		c.SendAllGames()
		c.SendAllRounds()
		c.SendConnectedPlayers()
		c.SendCurrentText()

		return
	}

	var player Player
	found := false

	for _, p := range players {
		if p.ID == payload.UUID {
			player = *p
			found = true

			p.Nickname = payload.Name
			c.PlayerID = &payload.UUID

			break
		}
	}

	if !found {
		player = Player{
			Nickname: payload.Name,
			ID:       uuid.New().String(),
		}

		c.PlayerID = &player.ID
		players = append(players, &player)

		c.SendSetUuid()
	}

	c.SendPlayerConnected(player)
	c.SendCurrentGame()
	c.SendAllAnswers()
	c.SendAllGames()
	c.SendAllRounds()
	c.SendConnectedPlayers()
	c.SendCurrentText()
}

func (c *Connection) SendSetUuid() {
	answer := SocketMessage{
		Type:    "set_uuid",
		Payload: *c.PlayerID,
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println("write:", err)
	}
}

func (c *Connection) SendPlayerConnected(player Player) {
	data, err := json.Marshal(player)
	if err != nil {
		return
	}

	msg := SocketMessage{
		Type:    "player_connected",
		Payload: string(data),
	}

	data, err = json.Marshal(msg)
	if err != nil {
		return
	}

	for _, conn := range connections {
		if c.PlayerID == nil {
			continue
		}

		if c.PlayerID == &player.ID {
			continue
		}

		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			fmt.Println("write:", err)
		}
	}
}

func (c *Connection) SetAnswer(msg SocketMessage) {
	player, err := c.GetPlayer()
	if err != nil {
		log.Println("get player:", err)
		return
	}

	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return
	}

	found := false
	for _, a := range answers {
		if a.GameID == round.GameID && a.PlayerID == player.ID && round.ID == a.RoundID {
			found = true
			a.Text = msg.Payload
			break
		}
	}

	if !found {
		newAnswer := Answer{
			GameID:   round.GameID,
			PlayerID: player.ID,
			RoundID:  round.ID,
			Text:     msg.Payload,
		}

		answers = append(answers, &newAnswer)

		for _, conn := range connections {
			if conn.PlayerID == nil {
				continue
			}

			conn.SendAllAnswers()
		}

		return
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		conn.SendAllAnswers()
	}
}

func (c *Connection) SetText(msg SocketMessage) {
	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return
	}

	for _, r := range rounds {
		if r.ID == round.ID {
			r.Question = msg.Payload
		}
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		conn.SendCurrentText()
	}
}

func (c *Connection) SendCurrentText() {
	if c.PlayerID == nil {
		return
	}

	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return
	}

	msg := SocketMessage{
		Type:    "set_text",
		Payload: round.Question,
	}

	data, err := json.Marshal(msg)

	if err != nil {
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))
	if err != nil {
		fmt.Printf("Failed to write message: %s\n", err)
	}
}

func (c *Connection) LeaveGame(msg SocketMessage) {
	player, err := c.GetPlayer()

	if err != nil {
		log.Println("get player:", err)
		return
	}

	for _, g := range games {
		if g.ID == msg.Payload {
			for i, p := range g.Players {
				if p == player.ID {
					g.Players = append(g.Players[:i], g.Players[i+1:]...)
					break
				}
			}
		}
	}

	answer := SocketMessage{
		Type:    "leave_game",
		Payload: player.ID,
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		if conn.PlayerID == c.PlayerID {
			continue
		}

		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			log.Println("write:", err)
		}
	}

	c.SendAllGames()
}

type SetCanTypePayload struct {
	CanType bool   `json:"canType"`
	GameID  string `json:"gameId"`
	RoundID string `json:"roundId"`
}

type AnswerPayload struct {
	Answer  string `json:"answer"`
	GameID  string `json:"gameId"`
	RoundID string `json:"roundId"`
}

func ChangeAnswerVisibility(msg SocketMessage, visible bool) {
	for _, a := range answers {
		if a.ID == msg.Payload {
			a.RevealedToPlayers = visible
			break
		}
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		conn.SendAllAnswers()
	}
}

func (c *Connection) HideAnswer(msg SocketMessage) {
	ChangeAnswerVisibility(msg, false)
}

func (c *Connection) RevealAnswer(msg SocketMessage) {
	ChangeAnswerVisibility(msg, true)
}

func (c *Connection) CreateGame(msg SocketMessage) {
	player, err := c.GetPlayer()

	if err != nil {
		log.Println("CREATE_GAME: get player:", err)
		return
	}

	game := Game{
		Name:          msg.Payload,
		Players:       []string{},
		ModeratorUUID: player.ID,
		ID:            uuid.New().String(),
	}

	round := GameRound{
		ID:       uuid.New().String(),
		GameID:   game.ID,
		Active:   true,
		Round:    1,
		Answers:  []Answer{},
		Question: "",
	}

	games = append(games, &game)
	rounds = append(rounds, &round)

	for _, g := range games {
		if g.ID != game.ID {
			for i, p := range g.Players {
				if p == player.ID {
					g.Players = append(g.Players[:i], g.Players[i+1:]...)
					break
				}
			}
		}
	}

	c.SendAllGames()
	c.SendAllRounds()
	c.SendCurrentGame()
	c.SendAllAnswers()
	c.SendCurrentText()
	c.SendConnectedPlayers()

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		if conn.PlayerID == c.PlayerID {
			continue
		}

		conn.SendAllGames()
		conn.SendAllRounds()
		conn.SendCurrentGame()
		conn.SendAllAnswers()
		conn.SendCurrentText()
		conn.SendConnectedPlayers()
	}
}

func (c *Connection) SendAllGames() {
	data, err := json.Marshal(games)
	if err != nil {
		log.Println("marshal:", err)
		return
	}

	response := SocketMessage{
		Type:    "get_games",
		Payload: string(data),
	}

	responseData, err := json.Marshal(response)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (c *Connection) SendAllRounds() {
	game, err := c.GetActiveGame()

	if err != nil {
		log.Println("get active game:", err)
		return
	}

	responseRounds := []GameRound{}

	for _, r := range rounds {
		if r.GameID == game.ID {
			responseRounds = append(responseRounds, *r)
		}
	}

	data, err := json.Marshal(responseRounds)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	answer := SocketMessage{
		Type:    "get_rounds",
		Payload: string(data),
	}

	responseData, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return
	}
}

type JoinGamePayload struct {
	GameID string `json:"gameId"`
}

func (c *Connection) GoNextRound(msg SocketMessage) {
	player, err := c.GetPlayer()

	if err != nil {
		return
	}

	var payload JoinGamePayload
	err = json.Unmarshal([]byte(msg.Payload), &payload)
	if err != nil {
		return
	}

	found := false
	nextRound := 1

	for _, g := range games {
		if g.ID == payload.GameID && g.ModeratorUUID == player.ID {
			for _, r := range rounds {
				if r.Active {
					r.Active = false
					r.Ended = true
					r.Started = false
					nextRound = r.Round + 1
					found = true
					break
				}
			}
		}
	}

	if !found {
		fmt.Println("Game or round not found")
		return
	}

	newRound := GameRound{
		GameID:   payload.GameID,
		Active:   true,
		Question: "",
		Answers:  []Answer{},
		Round:    nextRound,
		Started:  false,
		Ended:    false,
		ID:       uuid.New().String(),
	}

	rounds = append(rounds, &newRound)

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		g, err := conn.GetActiveGame()

		if err != nil {
			log.Println("get active game:", err)
			continue
		}

		if g.ID != payload.GameID {
			continue
		}

		conn.SendAllRounds()
		conn.SendAllGames()
		conn.SendConnectedPlayers()
		conn.SendAllAnswers()
		conn.SendCurrentText()
	}
}

func (c *Connection) SendCurrentGame() {
	game, err := c.GetActiveGame()

	if err != nil {
		log.Println("get active game:", err)
		return
	}

	data, err := json.Marshal(game)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	answer := SocketMessage{
		Type:    "get_game",
		Payload: string(data),
	}

	responseData, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (c *Connection) StartRound() {
	round, err := c.GetActiveRound()
	if err != nil {
		log.Println("get active round:", err)
		return
	}

	for _, r := range rounds {
		if r.ID == round.ID {
			r.Started = true
			r.Ended = false
		}
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		conn.SendAllRounds()
	}
}

func (c *Connection) EndRound() {
	round, err := c.GetActiveRound()
	if err != nil {
		log.Println("get active round:", err)
		return
	}

	for _, r := range rounds {
		if r.ID == round.ID {
			r.Ended = true
			r.Started = false
		}
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		conn.SendAllRounds()
	}
}

func (c *Connection) DeleteAnswer(msg SocketMessage) {
	for i, a := range answers {
		if a.ID == msg.Payload {
			answers = append(answers[:i], answers[i+1:]...)
			break
		}
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		conn.SendAllAnswers()
	}
}

func (c *Connection) DeleteGame(msg SocketMessage) {
	player, err := c.GetPlayer()
	if err != nil {
		log.Println("get player:", err)
		return
	}

	for i, g := range games {
		if g.ID == msg.Payload && g.ModeratorUUID == player.ID {
			games = append(games[:i], games[i+1:]...)
			break
		}
	}

	for i, a := range answers {
		if a.GameID == msg.Payload {
			answers = append(answers[:i], answers[i+1:]...)
			break
		}
	}

	msg = SocketMessage{
		Type:    "game_deleted",
		Payload: msg.Payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("marshal:", err)
		return
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err = conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))
		if err != nil {
			log.Println("write:", err)
			continue
		}

		conn.SendAllGames()
	}
}

func (c *Connection) Listen() {
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			c.Remove()
			break
		}

		msg, err := ParseSocketMessage(message)

		if err != nil {
			log.Println("parse:", err)
			c.Remove()
			break
		}

		switch msg.Type {
		case "create_game":
			c.CreateGame(msg)
		case "leave_game":
			c.LeaveGame(msg)
		case "get_text":
			c.SendCurrentText()
		case "join_game":
			c.JoinGame(msg)
		case "set_answer":
			c.SetAnswer(msg)
		case "set_text":
			c.SetText(msg)
		case "set_answer_visible":
			c.RevealAnswer(msg)
		case "set_answer_invisible":
			c.HideAnswer(msg)
		case "get_connected_players":
			c.SendConnectedPlayers()
		case "end_round":
			c.EndRound()
		case "start_round":
			c.StartRound()
		case "get_rounds":
			c.SendAllRounds()
		case "delete_answer":
			c.DeleteAnswer(msg)
		case "delete_game":
			c.DeleteGame(msg)
		case "go_next_round":
			c.GoNextRound(msg)
		case "get_game":
			c.SendCurrentGame()
		case "say_hello":
			c.SayHello(msg)
		default:
			c.UnhandledMessage(msg)
		}
	}
}
