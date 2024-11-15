package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitDatabase() (*mongo.Client, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return nil, fmt.Errorf("MONGODB_URI is not set")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().
		ApplyURI(uri))

	if err != nil {
		return nil, err
	}

	return client, nil
}

func DisconnectDatabase(client *mongo.Client) {
	if err := client.Disconnect(context.TODO()); err != nil {
		fmt.Println(err)
	}
}

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
