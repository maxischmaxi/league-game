package main

import (
	"fmt"

	"github.com/google/uuid"
)

func GetGame(id string) (*Game, error) {
	for _, game := range games {
		if game.ID == id {
			return &game, nil
		}
	}

	return nil, fmt.Errorf("Game with id %s not found", id)
}

func CreateGame(name string, creatorUuid string) string {
	gameId := uuid.New().String()

	game := Game{
		ID:            gameId,
		Name:          name,
		ModeratorUUID: creatorUuid,
	}

	games = append(games, game)

	return gameId
}
