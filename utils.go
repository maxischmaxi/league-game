package main

import (
	"encoding/json"
)

func ParseSocketMessage(data []byte) (SocketMessage, error) {
	var message SocketMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return SocketMessage{}, err
	}

	return message, nil
}
