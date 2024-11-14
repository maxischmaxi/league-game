package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Connection struct {
	Conn            *websocket.Conn
	UUID            string
	GameId          string
	Nick            string
	AnswerRevielead bool
	Answer          string
	IsModerator     bool
}

func (c *Connection) Remove() {
	p := Player{
		UUID:   c.UUID,
		Nick:   c.Nick,
		GameId: c.GameId,
	}

	player, err := json.Marshal(p)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	msg := SocketMessage{
		Type:    "player_disconnected",
		Payload: string(player),
	}

	data, err := json.Marshal(msg)

	if err != nil {
		log.Println("marshal:", err)
	}

	for i, conn := range connections {
		if conn.UUID == c.UUID {
			connections = append(connections[:i], connections[i+1:]...)
			continue
		}

		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			log.Println("write:", err)
		}
	}
}

func (c *Connection) GetConnectedUsers(msg SocketMessage) error {
	players := []Player{}
	for _, conn := range connections {
		if conn.UUID != c.UUID && conn.Nick != "" {
			players = append(players, Player{
				UUID:   conn.UUID,
				Nick:   conn.Nick,
				GameId: conn.GameId,
			})
		}
	}

	payload, err := json.Marshal(players)

	if err != nil {
		log.Println("marshal:", err)
		return err
	}

	answer := SocketMessage{
		Type:    "get_connected_users_response",
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

func (c *Connection) SetNick(msg SocketMessage) error {
	c.Nick = msg.Payload

	answer := SocketMessage{
		Type:    "nick_set_success",
		Payload: msg.Payload,
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

	log.Printf("Player %s connected with nickname %s", c.UUID, c.Nick)
	log.Printf("Remaining players: %v", len(connections))

	return nil
}

func (c *Connection) JoinGame(msg SocketMessage) error {
	found := false

	for _, game := range games {
		if game.ID == msg.Payload {
			found = true
			c.GameId = game.ID
			if game.ModeratorUUID == c.UUID {
				c.IsModerator = true
				log.Printf("Player %s is moderator of game %s", c.Nick, c.GameId)
			} else {
				c.IsModerator = false
				log.Printf("Player %s joined game %s", c.Nick, c.GameId)
			}
		}
	}

	if !found {
		log.Printf("Player %s tried to join non-existing game %s", c.Nick, msg.Payload)

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

	answer := SocketMessage{
		Type:    "join_game",
		Payload: msg.Payload,
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

	BroadcastPlayerConnected(*c)
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

func (c *Connection) SayHello(msg SocketMessage) error {
	c.UUID = msg.Payload
	if c.Nick == "" {
		answer := SocketMessage{
			Type:    "nick_request",
			Payload: "",
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
	}

	return nil
}

func (c *Connection) SetText(msg SocketMessage) error {
	var payload GameText
	err := json.Unmarshal([]byte(msg.Payload), &payload)

	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}

	found := false
	for i, gt := range game_texts {
		if gt.GameId == payload.GameId {
			game_texts[i].Text = payload.Text
			found = true
			break
		}
	}

	if !found {
		game_texts = append(game_texts, payload)
	}

	SendTextUpdateToAll(payload.GameId)
	SendCanTypeToAll()

	return nil
}

func (c *Connection) LeaveGame() {
	log.Printf("Player %s left game %s", c.Nick, c.GameId)
	c.GameId = ""
	c.IsModerator = false

	answer := SocketMessage{
		Type:    "leave_game",
		Payload: c.UUID,
	}

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	for _, conn := range connections {
		if conn.UUID == c.UUID {
			continue
		}

		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			log.Println("write:", err)
		}
	}
}

func (c *Connection) CanType(msg SocketMessage) {
	found := false

	for i, game := range allowd_games {
		if game == c.GameId {
			found = true

			if msg.Payload != "true" {
				allowd_games = append(allowd_games[:i], allowd_games[i+1:]...)
			}
		}
	}

	if !found && msg.Payload == "true" {
		allowd_games = append(allowd_games, c.GameId)
	}

	SendCanTypeToAll()
}

func (c *Connection) SetAnswer(msg SocketMessage) error {
	var payload Answer
	err := json.Unmarshal([]byte(msg.Payload), &payload)

	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}

	c.Answer = payload.Answer
	log.Printf("Player %s set answer for game %s", c.Nick, c.GameId)
	err = c.SendAllAnswersToModerator()

	if err != nil {
		log.Println("send all answers:", err)
		return err
	}

	return nil
}

func (c *Connection) GetText() error {
	if c.GameId == "" {
		log.Printf("Player %s requested text without game", c.Nick)
		return nil
	}

	log.Printf("Player %s requested text for game %s", c.Nick, c.GameId)

	for _, gt := range game_texts {
		if gt.GameId == c.GameId {
			log.Printf("Sending text to player %s for game %s", c.Nick, gt.GameId)
			payload, err := json.Marshal(gt)

			if err != nil {
				log.Println("marshal:", err)
				return err
			}

			answer := SocketMessage{
				Type:    "text_update",
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
				return err
			}

			return nil
		}
	}

	return nil
}

func (c *Connection) SendAllAnswersToModerator() error {
	allAnswers := []AllAnswer{}

	for _, conn := range connections {
		if !conn.IsModerator {
			log.Printf("Player %s is not moderator, adding answer %s", conn.Nick, conn.Answer)
			allAnswers = append(allAnswers, AllAnswer{
				UUID:   conn.UUID,
				Nick:   conn.Nick,
				Answer: conn.Answer,
			})
		}
	}

	allAnswerPayload, err := json.Marshal(allAnswers)

	if err != nil {
		log.Println("marshal:", err)
	}

	answer := SocketMessage{
		Type:    "all_answers",
		Payload: string(allAnswerPayload),
	}

	log.Printf("Sending %v answers to moderator", len(allAnswers))

	data, err := json.Marshal(answer)

	if err != nil {
		log.Println("marshal:", err)
	}

	for _, conn := range connections {
		if !conn.IsModerator {
			continue
		}

		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			log.Println("write:", err)
		}
	}

	return nil
}

func (c *Connection) Listen() {
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Print("read:", err)
			c.Remove()
			log.Printf("Player %s disconnected", c.UUID)
			log.Printf("Remaining players: %v", len(connections))
			break
		}

		msg, err := ParseSocketMessage(message)

		if err != nil {
			log.Println("parse:", err)
			c.Remove()
			break
		}

		switch msg.Type {
		case "leave_game":
			c.LeaveGame()
		case "get_text":
			err := c.GetText()

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

			err = c.SendAllAnswersToModerator()

			if err != nil {
				c.Remove()
				break
			}
		case "get_can_type":
			SendCanTypeToAll()
		case "can_type":
			c.CanType(msg)
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
		case "get_connected_users":
			err := c.GetConnectedUsers(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "say_hello":
			err := c.SayHello(msg)

			if err != nil {
				c.Remove()
				break
			}
		case "set_nick":
			err := c.SetNick(msg)

			if err != nil {
				c.Remove()
				break
			}
		default:
			c.UnhandledMessage(msg)
		}
	}
}
