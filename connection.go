package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Connection struct {
	Conn     *websocket.Conn
	PlayerID *primitive.ObjectID
}

func (c *Connection) GetPlayer(client *mongo.Client) (*Player, error) {
	if c.PlayerID == nil {
		return nil, fmt.Errorf("player id is nil")
	}

	var player Player

	coll := client.Database("league").Collection("players")
	filter := bson.M{"_id": c.PlayerID}

	err := coll.FindOne(context.TODO(), filter).Decode(&player)

	if err != nil {
		log.Println("find one:", err)
		return nil, err
	}

	return &player, nil
}

func (c *Connection) Remove(client *mongo.Client) {
	player, err := c.GetPlayer(client)

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

func (c *Connection) JoinGame(msg SocketMessage, client *mongo.Client) error {
	player, err := c.GetPlayer(client)
	if err != nil {
		log.Println("get player:", err)
		return err
	}

	coll := client.Database("league").Collection("games")
	id, err := primitive.ObjectIDFromHex(msg.Payload)
	if err != nil {
		log.Println("object id from hex:", err)
		return err
	}

	filter := bson.D{{Key: "_id", Value: id}}
	var game Game

	err = coll.FindOne(context.TODO(), filter).Decode(&game)
	if err != nil {
		if err == mongo.ErrNoDocuments {
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

		log.Println("find one:", err)
		return err
	}

	players := append(game.Players, *c.PlayerID)
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "players", Value: players}}}}

	_, err = coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("update one:", err)
		return err
	}

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

	data, err = json.Marshal(player)
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

		player, err := conn.GetPlayer(client)
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

	return c.SendAllAnswers(client)
}

func (c *Connection) SendAllAnswers(client *mongo.Client) error {
	game, err := c.GetActiveGame(client)
	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	round, err := c.GetActiveRound(client)
	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	coll := client.Database("league").Collection("answers")
	filter := bson.D{{Key: "gameId", Value: game.ID}, {Key: "roundId", Value: round.ID}}

	var answers []Answer

	cur, err := coll.Find(context.TODO(), filter)
	if err != nil {
		log.Println("find:", err)
		return err
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

func (c *Connection) GetActiveRound(client *mongo.Client) (*GameRound, error) {
	game, err := c.GetActiveGame(client)

	if err != nil {
		log.Println("get active game:", err)
		return nil, err
	}

	coll := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "active", Value: true}, {Key: "gameId", Value: game.ID}}

	var round GameRound

	err = coll.FindOne(context.TODO(), filter).Decode(&round)

	if err != nil {
		log.Println("find one:", err)
		return nil, err
	}

	return &round, nil
}

func (c *Connection) GetActiveGame(client *mongo.Client) (*Game, error) {
	player, err := c.GetPlayer(client)

	if err != nil {
		log.Println("GetActiveGame: get player:", err)
		return nil, err
	}

	coll := client.Database("league").Collection("games")
	filter := bson.D{{Key: "players", Value: bson.D{{Key: "$in", Value: []primitive.ObjectID{player.ID}}}}}
	var games []Game

	cur, err := coll.Find(context.TODO(), filter)

	if err != nil {
		log.Println("find:", err)
		return nil, err
	}

	for cur.Next(context.Background()) {
		var game Game
		err := cur.Decode(&game)

		if err != nil {
			log.Println("decode:", err)
			continue
		}

		games = append(games, game)
	}

	if len(games) == 0 {
		filter = bson.D{{Key: "moderatorId", Value: player.ID}}

		cur, err = coll.Find(context.TODO(), filter)

		if err != nil {
			log.Println("find:", err)
			return nil, err
		}

		for cur.Next(context.Background()) {
			var game Game
			err := cur.Decode(&game)

			if err != nil {
				log.Println("decode:", err)
				continue
			}

			games = append(games, game)
		}

		if len(games) == 0 {
			return nil, fmt.Errorf("no active games")
		}

		if len(games) > 1 {
			return nil, fmt.Errorf("multiple active games")
		}

		return &games[0], nil
	}

	if len(games) > 1 {
		return nil, fmt.Errorf("multiple active games")
	}

	return &games[0], nil
}

func (c *Connection) SendConnectedPlayers(client *mongo.Client) error {
	game, err := c.GetActiveGame(client)
	if err != nil {
		log.Println("SEND_CONNECTED_PLAYERS: get active game:", err)
		return err
	}

	coll := client.Database("league").Collection("players")
	filter := bson.D{}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		g, err := conn.GetActiveGame(client)

		if err != nil {
			log.Println("get active game:", err)
			continue
		}

		if game.ID != g.ID {
			continue
		}

		filter = append(filter, bson.E{Key: "_id", Value: conn.PlayerID})
	}

	cur, err := coll.Find(context.TODO(), filter)

	if err != nil {
		log.Println("find:", err)
		return err
	}

	var players []Player

	for cur.Next(context.Background()) {
		var player Player
		err := cur.Decode(&player)
		if err != nil {
			log.Println("decode:", err)
			return err
		}

		players = append(players, player)
	}

	if len(players) == 0 {
		players = []Player{}
	}

	payload, err := json.Marshal(players)

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

func (c *Connection) SayHello(msg SocketMessage, client *mongo.Client) error {
	var payload SayHelloPayload

	err := json.Unmarshal([]byte(msg.Payload), &payload)
	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}

	if payload.UUID != "" {
		playerId, err := primitive.ObjectIDFromHex(payload.UUID)

		if err != nil {
			log.Println("object id from hex:", err)
			return err
		}

		filter := bson.D{{Key: "_id", Value: playerId}}
		coll := client.Database("league").Collection("players")

		var player Player

		err = coll.FindOne(context.TODO(), filter).Decode(&player)

		if err != nil {
			log.Println("find one:", err)
			return err
		}

		player.Nickname = payload.Name

		update := bson.D{{Key: "$set", Value: bson.D{{Key: "nickname", Value: payload.Name}}}}
		_, err = coll.UpdateOne(context.TODO(), filter, update)

		if err != nil {
			log.Println("update one:", err)
			return err
		}

		c.PlayerID = &playerId

		gamesColl := client.Database("league").Collection("games")
		filter = bson.D{{Key: "players", Value: bson.D{{Key: "$in", Value: []primitive.ObjectID{playerId}}}}}

		var games []Game

		cur, err := gamesColl.Find(context.TODO(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil
			}

			log.Println("find:", err)
			return err
		}

		for cur.Next(context.Background()) {
			var game Game
			err := cur.Decode(&game)

			if err != nil {
				log.Println("decode:", err)
				continue
			}

			games = append(games, game)
		}

		if len(games) == 0 {
			return nil
		}

		if len(games) > 1 {
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

	coll := client.Database("league").Collection("players")
	id := primitive.NewObjectID()
	newPlayer := Player{
		Nickname: payload.Name,
		ID:       id,
	}

	_, err = coll.InsertOne(context.TODO(), newPlayer)
	if err != nil {
		log.Println("insert one:", err)
		return err
	}

	c.PlayerID = &id

	answer := SocketMessage{
		Type:    "set_uuid",
		Payload: id.Hex(),
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

func (c *Connection) SetAnswer(msg SocketMessage, client *mongo.Client) error {
	player, err := c.GetPlayer(client)
	if err != nil {
		log.Println("get player:", err)
		return err
	}

	round, err := c.GetActiveRound(client)

	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	answerColl := client.Database("league").Collection("answers")
	filter := bson.D{{Key: "gameId", Value: round.GameID}, {Key: "playerId", Value: player.ID}, {Key: "roundId", Value: round.ID}}

	var answer Answer
	err = answerColl.FindOne(context.TODO(), filter).Decode(&answer)

	if err == mongo.ErrNoDocuments {
		newAnswer := Answer{
			GameID:   round.GameID,
			PlayerID: player.ID,
			RoundID:  round.ID,
			Text:     msg.Payload,
		}

		_, err = answerColl.InsertOne(context.TODO(), newAnswer)

		if err != nil {
			log.Println("insert one:", err)
			return err
		}

		for _, conn := range connections {
			if conn.PlayerID == nil {
				continue
			}

			err := conn.SendAllAnswers(client)

			if err != nil {
				log.Println("send all answers:", err)
			}
		}

		return nil
	}

	if err != nil {
		log.Println("find one:", err)
		return err
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "text", Value: msg.Payload}}}}
	filter = bson.D{{Key: "_id", Value: answer.ID}}

	_, err = answerColl.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("update one:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendAllAnswers(client)

		if err != nil {
			log.Println("send all answers:", err)
		}
	}

	return nil
}

func (c *Connection) SetText(msg SocketMessage, client *mongo.Client) error {
	round, err := c.GetActiveRound(client)

	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	round.Question = msg.Payload

	coll := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "_id", Value: round.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "question", Value: msg.Payload}}}}

	_, err = coll.UpdateOne(context.TODO(), filter, update)

	if err != nil {
		log.Println("update one:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendCurrentText(client)

		if err != nil {
			log.Println("send current text:", err)
		}
	}

	return nil
}

func (c *Connection) SendCurrentText(client *mongo.Client) error {
	if c.PlayerID == nil {
		return fmt.Errorf("player id is nil")
	}

	round, err := c.GetActiveRound(client)

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

func (c *Connection) LeaveGame(msg SocketMessage, client *mongo.Client) error {
	gameId, err := primitive.ObjectIDFromHex(msg.Payload)

	if err != nil {
		log.Println("object id from hex:", err)
		return err
	}

	player, err := c.GetPlayer(client)

	if err != nil {
		log.Println("get player:", err)
		return err
	}

	coll := client.Database("league").Collection("games")
	filter := bson.D{{Key: "_id", Value: gameId}}
	var game Game

	err = coll.FindOne(context.TODO(), filter).Decode(&game)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("Player %s tried to leave non-existing game %s", player.Nickname, gameId)
			return nil
		}

		log.Println("find one:", err)
		return err
	}

	for i, p := range game.Players {
		if p == player.ID {
			game.Players = append(game.Players[:i], game.Players[i+1:]...)
			break
		}
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "players", Value: game.Players}}}}

	_, err = coll.UpdateOne(context.TODO(), filter, update)

	if err != nil {
		log.Println("update one:", err)
		return err
	}

	log.Printf("Player %s left game", player.Nickname)

	answer := SocketMessage{
		Type:    "leave_game",
		Payload: player.ID.Hex(),
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

	gamesColl := client.Database("league").Collection("games")
	cur, err := gamesColl.Find(context.TODO(), bson.D{})

	if err != nil {
		log.Println("find:", err)
		return err
	}

	var games []Game

	for cur.Next(context.Background()) {
		var game Game
		err := cur.Decode(&game)

		if err != nil {
			log.Println("decode:", err)
			continue
		}

		games = append(games, game)
	}

	if len(games) == 0 {
		games = []Game{}
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

func ChangeAnswerVisibility(msg SocketMessage, client *mongo.Client, visible bool) error {
	answerId, err := primitive.ObjectIDFromHex(msg.Payload)
	if err != nil {
		log.Println("object id from hex:", err)
		return err
	}

	answerColl := client.Database("league").Collection("answers")
	filter := bson.D{{Key: "_id", Value: answerId}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "revealedToPlayers", Value: visible}}}}
	_, err = answerColl.UpdateOne(context.TODO(), filter, update)

	if err != nil {
		log.Println("update one:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendAllAnswers(client)

		if err != nil {
			log.Println("send all answers:", err)
		}
	}

	return nil
}

func (c *Connection) HideAnswer(msg SocketMessage, client *mongo.Client) error {
	return ChangeAnswerVisibility(msg, client, false)
}

func (c *Connection) RevealAnswer(msg SocketMessage, client *mongo.Client) error {
	return ChangeAnswerVisibility(msg, client, true)
}

func (c *Connection) CreateGame(msg SocketMessage, client *mongo.Client) error {
	player, err := c.GetPlayer(client)

	if err != nil {
		log.Println("CREATE_GAME: get player:", err)
		return err
	}

	coll := client.Database("league").Collection("games")

	game := Game{
		Name:          msg.Payload,
		Players:       []primitive.ObjectID{},
		ModeratorUUID: player.ID,
		ID:            primitive.NewObjectID(),
	}

	result, err := coll.InsertOne(context.TODO(), game)

	if err != nil {
		log.Println("insert one:", err)
		return err
	}

	round := GameRound{
		ID:       primitive.NewObjectID(),
		GameID:   result.InsertedID.(primitive.ObjectID),
		Active:   true,
		Round:    1,
		Answers:  []Answer{},
		Question: "",
	}

	roundsColl := client.Database("league").Collection("rounds")

	_, err = roundsColl.InsertOne(context.TODO(), round)

	if err != nil {
		log.Println("insert one:", err)
		return err
	}

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

		err = c.SendAllGames(client)

		if err != nil {
			log.Println("send all games:", err)
		}

		err := conn.SendRound(client)

		if err != nil {
			log.Println("send round:", err)
		}
	}

	return nil
}

func (c *Connection) SendRound(client *mongo.Client) error {
	game, err := c.GetActiveGame(client)
	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	coll := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "active", Value: true}, {Key: "gameId", Value: game.ID}}

	var round GameRound

	err = coll.FindOne(context.TODO(), filter).Decode(&round)

	if err != nil {
		log.Println("find one:", err)
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

func (c *Connection) SendAllGames(client *mongo.Client) error {
	coll := client.Database("league").Collection("games")
	var games []Game

	cur, err := coll.Find(context.TODO(), bson.D{})

	if err != nil {
		log.Println("find:", err)
		return err
	}

	for cur.Next(context.Background()) {
		var game Game
		err := cur.Decode(&game)

		if err != nil {
			log.Println("decode:", err)
			continue
		}

		games = append(games, game)
	}

	if len(games) == 0 {
		games = []Game{}
	}

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

func (c *Connection) SendAllRounds(client *mongo.Client) error {
	game, err := c.GetActiveGame(client)

	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	roundsColl := client.Database("league").Collection("rounds")

	filter := bson.D{{Key: "gameId", Value: game.ID}}

	var rounds []GameRound

	cur, err := roundsColl.Find(context.TODO(), filter)

	if err != nil {

		log.Println("find:", err)
		return err
	}

	for cur.Next(context.Background()) {
		var round GameRound
		err := cur.Decode(&round)

		if err != nil {
			log.Println("decode:", err)
			continue
		}

		rounds = append(rounds, round)
	}

	if len(rounds) == 0 {
		rounds = []GameRound{}
	}

	data, err := json.Marshal(rounds)

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

func (c *Connection) SendCurrentRound(client *mongo.Client) error {
	game, err := c.GetActiveGame(client)

	if err != nil {
		log.Println("get active game:", err)
		return err
	}

	roundsColl := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "active", Value: true}, {Key: "gameId", Value: game.ID}}

	var round GameRound

	err = roundsColl.FindOne(context.TODO(), filter).Decode(&round)

	if err != nil {
		log.Println("find one:", err)
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

func (c *Connection) GoNextRound(msg SocketMessage, client *mongo.Client) error {
	player, err := c.GetPlayer(client)

	if err != nil {
		return err
	}

	var payload JoinGamePayload

	err = json.Unmarshal([]byte(msg.Payload), &payload)

	if err != nil {
		return err
	}

	gameId, err := primitive.ObjectIDFromHex(payload.GameID)
	if err != nil {
		return err
	}

	roundId, err := primitive.ObjectIDFromHex(payload.RoundID)
	if err != nil {
		return err
	}

	gamesColl := client.Database("league").Collection("games")
	filter := bson.D{{Key: "_id", Value: gameId}}

	var game Game
	err = gamesColl.FindOne(context.TODO(), filter).Decode(&game)
	if err != nil {
		return err
	}

	if game.ModeratorUUID != player.ID {
		return fmt.Errorf("player is not moderator")
	}

	roundsColl := client.Database("league").Collection("rounds")
	filter = bson.D{{Key: "_id", Value: roundId}}

	var round GameRound
	err = roundsColl.FindOne(context.TODO(), filter).Decode(&round)
	if err != nil {
		return err
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "active", Value: false}, {Key: "ended", Value: true}, {Key: "started", Value: false}}}}
	filter = bson.D{{Key: "_id", Value: roundId}}

	_, err = roundsColl.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	newRound := GameRound{
		GameID:   gameId,
		Active:   true,
		Question: "",
		Answers:  []Answer{},
		Round:    round.Round + 1,
		Started:  false,
		Ended:    false,
	}

	_, err = roundsColl.InsertOne(context.TODO(), newRound)

	if err != nil {
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		g, err := conn.GetActiveGame(client)

		if err != nil {
			log.Println("get active game:", err)
			continue
		}

		if g.ID != gameId {
			continue
		}

		err = conn.SendAllRounds(client)
		if err != nil {
			log.Println("send all rounds:", err)
			continue
		}

		err = conn.SendCurrentRound(client)
		if err != nil {
			log.Println("send current round:", err)
			continue
		}

		err = conn.SendAllGames(client)
		if err != nil {
			log.Println("send all games:", err)
			continue
		}

		err = conn.SendConnectedPlayers(client)
		if err != nil {
			log.Println("send connected users:", err)
		}

		err = conn.SendAllAnswers(client)
		if err != nil {
			log.Println("send all answers:", err)
		}

		err = conn.SendRoundState(client)
		if err != nil {
			log.Println("send round state:", err)
		}

		err = conn.SendCurrentText(client)
		if err != nil {
			log.Println("send current text:", err)
		}
	}

	return nil
}

func (c *Connection) SendCurrentGame(client *mongo.Client) error {
	game, err := c.GetActiveGame(client)

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

func (c *Connection) SendRoundState(client *mongo.Client) error {
	round, err := c.GetActiveRound(client)

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

func (c *Connection) StartRound(client *mongo.Client) error {
	round, err := c.GetActiveRound(client)
	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	coll := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "_id", Value: round.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "started", Value: true}, {Key: "ended", Value: false}}}}

	_, err = coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("update one:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendRoundState(client)

		if err != nil {
			log.Println("send round state:", err)
		}
	}

	return nil
}

func (c *Connection) EndRound(client *mongo.Client) error {
	round, err := c.GetActiveRound(client)
	if err != nil {
		log.Println("get active round:", err)
		return err
	}

	coll := client.Database("league").Collection("rounds")
	filter := bson.D{{Key: "_id", Value: round.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "started", Value: false}, {Key: "ended", Value: true}}}}

	_, err = coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("update one:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendRoundState(client)

		if err != nil {
			log.Println("send round state:", err)
		}
	}

	return nil
}

func (c *Connection) DeleteAnswer(msg SocketMessage, client *mongo.Client) error {
	answerId, err := primitive.ObjectIDFromHex(msg.Payload)
	if err != nil {
		log.Println("object id from hex:", err)
		return err
	}

	answerColl := client.Database("league").Collection("answers")
	filter := bson.D{{Key: "_id", Value: answerId}}

	_, err = answerColl.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Println("delete one:", err)
		return err
	}

	for _, conn := range connections {
		if conn.PlayerID == nil {
			continue
		}

		err := conn.SendAllAnswers(client)

		if err != nil {
			log.Println("send all answers:", err)
		}
	}

	return nil
}

func (c *Connection) DeleteGame(msg SocketMessage, client *mongo.Client) error {
	gameId, err := primitive.ObjectIDFromHex(msg.Payload)
	if err != nil {
		log.Println("object id from hex:", err)
		return err
	}

	player, err := c.GetPlayer(client)
	if err != nil {
		log.Println("get player:", err)
		return err
	}

	coll := client.Database("league").Collection("games")
	filter := bson.D{{Key: "_id", Value: gameId}}

	var game Game
	err = coll.FindOne(context.TODO(), filter).Decode(&game)
	if err != nil {
		log.Println("find one:", err)
		return err
	}

	if game.ModeratorUUID != player.ID {
		return fmt.Errorf("player is not moderator")
	}

	_, err = coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Println("delete one:", err)
		return err
	}

	roundsColl := client.Database("league").Collection("rounds")
	filter = bson.D{{Key: "gameId", Value: gameId}}

	_, err = roundsColl.DeleteMany(context.TODO(), filter)
	if err != nil {
		log.Println("delete many:", err)
		return err
	}

	answerColl := client.Database("league").Collection("answers")
	filter = bson.D{{Key: "gameId", Value: gameId}}

	_, err = answerColl.DeleteMany(context.TODO(), filter)
	if err != nil {
		log.Println("delete many:", err)
		return err
	}

	msg = SocketMessage{
		Type:    "game_deleted",
		Payload: gameId.Hex(),
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

		err := conn.SendAllGames(client)
		if err != nil {
			log.Println("send all games:", err)
		}
	}

	return nil
}

func (c *Connection) Listen(client *mongo.Client) {
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			c.Remove(client)
			break
		}

		msg, err := ParseSocketMessage(message)

		if err != nil {
			log.Println("parse:", err)
			c.Remove(client)
			break
		}

		switch msg.Type {
		case "create_game":
			err := c.CreateGame(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "leave_game":
			err := c.LeaveGame(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "get_text":
			err := c.SendCurrentText(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "get_round":
			err := c.SendRound(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "join_game":
			err := c.JoinGame(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "set_answer":
			err := c.SetAnswer(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "set_text":
			err := c.SetText(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "set_answer_visible":
			err := c.RevealAnswer(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "set_answer_invisible":
			err := c.HideAnswer(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "get_connected_players":
			err := c.SendConnectedPlayers(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "end_round":
			err := c.EndRound(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "start_round":
			err := c.StartRound(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "get_rounds":
			err := c.SendAllRounds(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "delete_answer":
			err := c.DeleteAnswer(msg, client)
			if err != nil {
				c.Remove(client)
				break
			}
		case "delete_game":
			err := c.DeleteGame(msg, client)
			if err != nil {
				c.Remove(client)
				break
			}
		case "go_next_round":
			err := c.GoNextRound(msg, client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "get_game":
			err := c.SendCurrentGame(client)

			if err != nil {
				c.Remove(client)
				break
			}
		case "say_hello":
			err := c.SayHello(msg, client)
			if err != nil {
				log.Println("say hello:", err)
			}

			err = c.SendAllGames(client)
			if err != nil {
				log.Println("send all games:", err)
			}

			err = c.SendConnectedPlayers(client)
			if err != nil {
				log.Println("send connected users:", err)
			}

			err = c.SendAllRounds(client)
			if err != nil {
				log.Println("send all rounds:", err)
			}

			err = c.SendCurrentRound(client)
			if err != nil {
				log.Println("send current round:", err)
			}

			err = c.SendCurrentText(client)
			if err != nil {
				log.Println("send current text:", err)
			}

			err = c.SendCurrentGame(client)
			if err != nil {
				log.Println("send current game:", err)
			}

			err = c.SendAllAnswers(client)
			if err != nil {
				log.Println("send all answers:", err)
			}
		default:
			c.UnhandledMessage(msg)
		}
	}
}
