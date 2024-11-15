package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func SendTextUpdateToAll(gameId string) {
	for _, gt := range game_texts {
		if gt.GameId == gameId {
			payload, err := json.Marshal(gt)

			if err != nil {
				log.Println("marshal:", err)
				return
			}

			data := SocketMessage{
				Type:    "text_update",
				Payload: string(payload),
			}

			msg, err := json.Marshal(data)

			if err != nil {
				log.Println("marshal:", err)
				return
			}

			for _, conn := range connections {
				err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(msg))

				if err != nil {
					log.Println("write:", err)
				}
			}
		}
	}
}

func SendCanTypeToAll() {
	payload, err := json.Marshal(allowed_games)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	msg := SocketMessage{
		Type:    "can_type",
		Payload: string(payload),
	}

	data, err := json.Marshal(msg)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	for _, conn := range connections {
		err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

		if err != nil {
			log.Println("write:", err)
		}
	}
}

func BroadcastPlayerConnected(player Connection) {
	msg := Player{
		UUID:   player.UUID,
		Nick:   player.Nick,
		GameId: player.GameId,
	}

	p, err := json.Marshal(msg)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	message := SocketMessage{
		Type:    "player_connected",
		Payload: string(p),
	}

	data, err := json.Marshal(message)

	if err != nil {
		log.Println("marshal:", err)
		return
	}

	for _, conn := range connections {
		if conn.UUID != player.UUID {
			err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(data))

			if err != nil {
				log.Println("write:", err)
			}
		}
	}
}

func ParseSocketMessage(data []byte) (SocketMessage, error) {
	var message SocketMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return SocketMessage{}, err
	}

	return message, nil
}
