package main

import (
	"database/sql"
	"fmt"
)

type Database struct {
	db *sql.DB
}

type Game struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ModeratorUUID string `json:"uuid"`
}

func InitDatabase() (*Database, error) {
	db, err := sql.Open("sqlite3", "./game.db")

	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS games (id INTEGER PRIMARY KEY, name TEXT)")

	if err != nil {
		return nil, err
	}

	_, err = db.Exec("ALTER TABLE games ADD COLUMN uuid TEXT")

	if err != nil {
		fmt.Println("uuid column already exists")
	}

	return &Database{db: db}, nil
}

func (d *Database) GetGame(id string) (*Game, error) {
	row := d.db.QueryRow("SELECT id, name, uuid FROM games WHERE id = ?", id)

	var game Game

	err := row.Scan(&game.ID, &game.Name, &game.ModeratorUUID)

	if err != nil {
		return nil, err
	}

	return &game, nil
}

func (d *Database) CreateGame(name string, uuid string) (string, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return "", err
	}

	stmt, err := tx.Prepare("INSERT INTO games (name, uuid) VALUES (?, ?)")
	if err != nil {
		return "", err
	}

	res, err := stmt.Exec(name, uuid)
	if err != nil {
		return "", err
	}

	lastInserted, err := res.LastInsertId()

	if err != nil {
		return "", err
	}

	err = tx.Commit()

	return fmt.Sprintf("%d", lastInserted), err
}

func (d *Database) GetAllGames() ([]Game, error) {
	rows, err := d.db.Query("SELECT id, name, uuid FROM games")

	if err != nil {
		return nil, err
	}

	var games []Game

	for rows.Next() {
		var game Game

		err = rows.Scan(&game.ID, &game.Name, &game.ModeratorUUID)

		if err != nil {
			return nil, err
		}

		games = append(games, game)
	}

	return games, nil
}
