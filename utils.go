package main

import (
	"encoding/json"
	"fmt"
)

func ParseSocketMessage(data []byte) (SocketMessage, error) {
	var message SocketMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return SocketMessage{}, err
	}

	return message, nil
}

func UpdateGamePlayers(id string, players []string) error {
	for _, g := range games {
		if g.ID == id {
			g.Players = players
			return nil
		}
	}
	return fmt.Errorf("game not found")
}

func FindGameById(id string) (*Game, error) {
	for _, g := range games {
		if g.ID == id {
			return g, nil
		}
	}

	return nil, fmt.Errorf("game not found")
}

func FindAllAnswersByGameAndRound(gameId string, roundId string) (*[]Answer, error) {
	res := []Answer{}

	for _, a := range answers {
		if a.GameID == gameId && a.RoundID == roundId {
			res = append(res, *a)
		}
	}

	return &res, nil
}

func FindActiveRoundByGameId(gameId string) (*GameRound, error) {
	for _, r := range rounds {
		if r.GameID == gameId && r.Active {
			return r, nil
		}
	}

	return nil, fmt.Errorf("round not found")
}

func Parse[T any](data []byte) (*T, error) {
	var msg T

	err := json.Unmarshal(data, &msg)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}
