package main

import (
	"context"
	"encoding/json"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func ParseSocketMessage(data []byte) (SocketMessage, error) {
	var message SocketMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return SocketMessage{}, err
	}

	return message, nil
}

func UpdateGamePlayers(id primitive.ObjectID, players []primitive.ObjectID, client *mongo.Client) error {
	coll := client.Database("league").Collection("games")
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "players", Value: players}}}}
	filter := bson.D{{Key: "_id", Value: id}}

	_, err := coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("update one:", err)
		return err
	}

	return nil
}

func FindGameById(id primitive.ObjectID, client *mongo.Client) (*Game, error) {
	coll := client.Database("league").Collection("games")
	filter := bson.D{{Key: "_id", Value: id}}

	var game Game
	err := coll.FindOne(context.TODO(), filter).Decode(&game)

	if err != nil {
		return nil, err
	}

	return &game, nil
}

func FindAllAnswersByGameAndRound(gameId primitive.ObjectID, roundId primitive.ObjectID, client *mongo.Client) (*[]Answer, error) {
	coll := client.Database("league").Collection("answers")
	filter := bson.D{{Key: "gameId", Value: gameId}, {Key: "roundId", Value: roundId}}

	var answers []Answer

	cur, err := coll.Find(context.TODO(), filter)
	if err != nil {
		log.Println("find:", err)
		return nil, err
	}

	for cur.Next(context.Background()) {
		var answer Answer
		err := cur.Decode(&answer)

		if err != nil {
			log.Println("decode:", err)
			continue
		}

		answers = append(answers, answer)
	}

	if len(answers) == 0 {
		answers = []Answer{}
	}

	return &answers, nil
}

func FindActiveRoundByGameId(gameId primitive.ObjectID, client *mongo.Client) (*GameRound, error) {
	coll := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "active", Value: true}, {Key: "gameId", Value: gameId}}
	var round GameRound

	err := coll.FindOne(context.TODO(), filter).Decode(&round)
	if err != nil {
		return nil, err
	}

	return &round, nil
}
