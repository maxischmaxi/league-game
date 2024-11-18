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

func (c *Connection) JoinGame(msg SocketMessage) error {
	player, err := c.GetPlayer()
	if err != nil {
		log.Println("get player:", err)
		return err
	}

	game, err := FindGameById(msg.Payload)

	if err != nil {
		log.Println("find one:", err)
		return err
	}

	players := append(game.Players, *c.PlayerID)
	err = UpdateGamePlayers(game.ID, players)
	if err != nil {
		log.Println("update game players:", err)
		return err
	}

	err = c.SendJoinSuccess(*game)
	if err != nil {
		log.Println("send join success:", err)
		return err
	}

	err = c.SendPlayerConnectedToAll(*game, *player)
	if err != nil {
		log.Println("send player connected to all:", err)
		return err
	}

	return c.SendAllAnswers()
}

func (c *Connection) SendAllAnswers() error {
	game, err := c.GetActiveGame()
	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	round, err := c.GetActiveRound()
	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	answers, err := FindAllAnswersByGameAndRound(game.ID, round.ID)
	if err != nil {
		log.Println("find all answers by game and round:", err)
		return err
	}

	data, err := json.Marshal(answers)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	response := SocketMessage{
		Type:    "all_answers",
		Payload: string(data),
	}

	responseData, err := json.Marshal(response)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
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

func (c *Connection) SendConnectedPlayers() error {
	game, err := c.GetActiveGame()
	if err != nil {
		log.Println("SEND_CONNECTED_PLAYERS: get active game:", err)
		return err
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
		return err
	}

	answer := SocketMessage{
		Type:    "get_connected_players",
		Payload: string(payload),
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println("write:", err)
	}

	return nil
}

type SayHelloPayload struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

func (c *Connection) SayHello(msg SocketMessage) error {
	var payload SayHelloPayload

	err := json.Unmarshal([]byte(msg.Payload), &payload)
	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}

	if payload.UUID != "" {
		var player Player

		for _, p := range players {
			if p.ID == payload.UUID {
				p.Nickname = payload.Name
				c.PlayerID = &payload.UUID
				player = *p
				break
			}
		}

		if player.ID == "" {
			return fmt.Errorf("player not found")
		}

		playerGames := []Game{}
		for _, g := range games {
			for _, p := range g.Players {
				if p == payload.UUID {
					playerGames = append(playerGames, *g)
				}
			}
		}

		if len(playerGames) == 0 {
			return nil
		}

		if len(playerGames) > 1 {
			return nil
		}

		data, err := json.Marshal(player)
		if err != nil {
			log.Println("marshal:", err)
			return err
		}

		msg := SocketMessage{
			Type:    "player_connected",
			Payload: string(data),
		}

		data, err = json.Marshal(msg)

		if err != nil {
			log.Println("marshal:", err)
			return err
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

		return nil
	}

	newPlayer := Player{
		Nickname: payload.Name,
		ID:       uuid.New().String(),
	}

	players = append(players, &newPlayer)

	c.PlayerID = &newPlayer.ID

	answer := SocketMessage{
		Type:    "set_uuid",
		Payload: newPlayer.ID,
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println("write:", err)
	}

	return nil
}

func (c *Connection) SetAnswer(msg SocketMessage) error {
	player, err := c.GetPlayer()
	if err != nil {
		log.Println("get player:", err)
		return err
	}

	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return err
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

			err := conn.SendAllAnswers()

			if err != nil {
				log.Println("send all answers:", err)
			}
		}

		return nil
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendAllAnswers()

		if err != nil {
			log.Println("send all answers:", err)
		}
	}

	return nil
}

func (c *Connection) SetText(msg SocketMessage) error {
	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return err
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

		err := conn.SendCurrentText()

		if err != nil {
			log.Println("send current text:", err)
		}
	}

	return nil
}

func (c *Connection) SendCurrentText() error {
	if c.PlayerID == nil {
		return fmt.Errorf("player id is nil")
	}

	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	msg := SocketMessage{
		Type:    "set_text",
		Payload: round.Question,
	}

	data, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	return c.Conn.WriteMessage(websocket.TextMessage, []byte(data))
}

func (c *Connection) LeaveGame(msg SocketMessage) error {
	player, err := c.GetPlayer()

	if err != nil {
		log.Println("get player:", err)
		return err
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
		return err
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

	data, err = json.Marshal(games)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer = SocketMessage{
		Type:    "get_games",
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

func ChangeAnswerVisibility(msg SocketMessage, visible bool) error {
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

		err := conn.SendAllAnswers()

		if err != nil {
			log.Println("send all answers:", err)
		}
	}

	return nil
}

func (c *Connection) HideAnswer(msg SocketMessage) error {
	return ChangeAnswerVisibility(msg, false)
}

func (c *Connection) RevealAnswer(msg SocketMessage) error {
	return ChangeAnswerVisibility(msg, true)
}

func (c *Connection) CreateGame(msg SocketMessage) error {
	player, err := c.GetPlayer()

	if err != nil {
		log.Println("CREATE_GAME: get player:", err)
		return err
	}

	game := Game{
		Name:          msg.Payload,
		Players:       []string{},
		ModeratorUUID: player.ID,
		ID:            uuid.New().String(),
	}

	games = append(games, &game)

	round := GameRound{
		ID:       uuid.New().String(),
		GameID:   game.ID,
		Active:   true,
		Round:    1,
		Answers:  []Answer{},
		Question: "",
	}

	rounds = append(rounds, &round)

	data, err := json.Marshal(game)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer := SocketMessage{
		Type:    "create_game",
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

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err = c.SendAllGames()

		if err != nil {
			log.Println("send all games:", err)
		}

		err := conn.SendRound()

		if err != nil {
			log.Println("send round:", err)
		}
	}

	return nil
}

func (c *Connection) SendRound() error {
	game, err := c.GetActiveGame()
	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	for _, r := range rounds {
		if r.GameID == game.ID && r.Active {
			data, err := json.Marshal(r)
			if err != nil {
				log.Println("marshal:", err)
				return err
			}

			answer := SocketMessage{
				Type:    "get_round",
				Payload: string(data),
			}

			responseData, err := json.Marshal(answer)

			if err != nil {
				log.Println("marshal:", err)
				return err
			}

			err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

			if err != nil {
				log.Println("write:", err)
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("round not found")
}

func (c *Connection) SendAllGames() error {
	data, err := json.Marshal(games)
	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	response := SocketMessage{
		Type:    "get_games",
		Payload: string(data),
	}

	responseData, err := json.Marshal(response)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

func (c *Connection) SendAllRounds() error {
	game, err := c.GetActiveGame()

	if err != nil {
		log.Println("get active game:", err)
		return err
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
		return err
	}

	answer := SocketMessage{
		Type:    "get_rounds",
		Payload: string(data),
	}

	responseData, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

func (c *Connection) SendCurrentRound() error {
	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	data, err := json.Marshal(round)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer := SocketMessage{
		Type:    "get_round",
		Payload: string(data),
	}

	responseData, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

type JoinGamePayload struct {
	GameID  string `json:"gameId"`
	RoundID string `json:"roundId"`
}

func (c *Connection) GoNextRound(msg SocketMessage) error {
	player, err := c.GetPlayer()

	if err != nil {
		return err
	}

	var payload JoinGamePayload

	err = json.Unmarshal([]byte(msg.Payload), &payload)

	if err != nil {
		return err
	}

	found := false
	nextRound := 1

	for _, g := range games {
		if g.ID == payload.GameID && g.ModeratorUUID == player.ID {
			for _, r := range rounds {
				if r.ID == payload.RoundID {
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
		return fmt.Errorf("game not found")
	}

	newRound := GameRound{
		GameID:   payload.GameID,
		Active:   true,
		Question: "",
		Answers:  []Answer{},
		Round:    nextRound,
		Started:  false,
		Ended:    false,
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

		err = conn.SendAllRounds()
		if err != nil {
			log.Println("send all rounds:", err)
			continue
		}

		err = conn.SendCurrentRound()
		if err != nil {
			log.Println("send current round:", err)
			continue
		}

		err = conn.SendAllGames()
		if err != nil {
			log.Println("send all games:", err)
			continue
		}

		err = conn.SendConnectedPlayers()
		if err != nil {
			log.Println("send connected users:", err)
		}

		err = conn.SendAllAnswers()
		if err != nil {
			log.Println("send all answers:", err)
		}

		err = conn.SendRoundState()
		if err != nil {
			log.Println("send round state:", err)
		}

		err = conn.SendCurrentText()
		if err != nil {
			log.Println("send current text:", err)
		}
	}

	return nil
}

func (c *Connection) SendCurrentGame() error {
	game, err := c.GetActiveGame()

	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	data, err := json.Marshal(game)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer := SocketMessage{
		Type:    "get_game",
		Payload: string(data),
	}

	responseData, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

func (c *Connection) SendRoundState() error {
	round, err := c.GetActiveRound()

	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	data, err := json.Marshal(round)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer := SocketMessage{
		Type:    "get_round",
		Payload: string(data),
	}

	responseData, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, []byte(responseData))

	if err != nil {
		log.Println("write:", err)
		return err
	}

	return nil
}

func (c *Connection) StartRound() error {
	round, err := c.GetActiveRound()
	if err != nil {
		log.Println("get active round:", err)
		return err
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

		err := conn.SendRoundState()

		if err != nil {
			log.Println("send round state:", err)
		}
	}

	return nil
}

func (c *Connection) EndRound() error {
	round, err := c.GetActiveRound()
	if err != nil {
		log.Println("get active round:", err)
		return err
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

		err := conn.SendRoundState()

		if err != nil {
			log.Println("send round state:", err)
		}
	}

	return nil
}

func (c *Connection) DeleteAnswer(msg SocketMessage) error {
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

		err := conn.SendAllAnswers()

		if err != nil {
			log.Println("send all answers:", err)
		}
	}

	return nil
}

func (c *Connection) DeleteGame(msg SocketMessage) error {
	player, err := c.GetPlayer()
	if err != nil {
		log.Println("get player:", err)
		return err
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
		return err
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

		err := conn.SendAllGames()
		if err != nil {
			log.Println("send all games:", err)
		}
	}

	return nil
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
			err := c.CreateGame(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "leave_game":
			err := c.LeaveGame(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "get_text":
			err := c.SendCurrentText()

			if err != nil {
				c.Remove()
				break
			}
		case "get_round":
			err := c.SendRound()

			if err != nil {
				c.Remove()
				break
			}
		case "join_game":
			err := c.JoinGame(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "set_answer":
			err := c.SetAnswer(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "set_text":
			err := c.SetText(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "set_answer_visible":
			err := c.RevealAnswer(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "set_answer_invisible":
			err := c.HideAnswer(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "get_connected_players":
			err := c.SendConnectedPlayers()

			if err != nil {
				c.Remove()
				break
			}
		case "end_round":
			err := c.EndRound()

			if err != nil {
				c.Remove()
				break
			}
		case "start_round":
			err := c.StartRound()

			if err != nil {
				c.Remove()
				break
			}
		case "get_rounds":
			err := c.SendAllRounds()

			if err != nil {
				c.Remove()
				break
			}
		case "delete_answer":
			err := c.DeleteAnswer(msg)
			if err != nil {
				c.Remove()
				break
			}
		case "delete_game":
			err := c.DeleteGame(msg)
			if err != nil {
				c.Remove()
				break
			}
		case "go_next_round":
			err := c.GoNextRound(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "get_game":
			err := c.SendCurrentGame()

			if err != nil {
				c.Remove()
				break
			}
		case "say_hello":
			err := c.SayHello(msg)
			if err != nil {
				log.Println("say hello:", err)
			}

			err = c.SendAllGames()
			if err != nil {
				log.Println("send all games:", err)
			}

			err = c.SendConnectedPlayers()
			if err != nil {
				log.Println("send connected users:", err)
			}

			err = c.SendAllRounds()
			if err != nil {
				log.Println("send all rounds:", err)
			}

			err = c.SendCurrentRound()
			if err != nil {
				log.Println("send current round:", err)
			}

			err = c.SendCurrentText()
			if err != nil {
				log.Println("send current text:", err)
			}

			err = c.SendCurrentGame()
			if err != nil {
				log.Println("send current game:", err)
			}

			err = c.SendAllAnswers()
			if err != nil {
				log.Println("send all answers:", err)
			}
		default:
			c.UnhandledMessage(msg)
		}
	}
}
