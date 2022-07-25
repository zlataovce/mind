package main

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	*sql.DB
	path string
}

type Data struct {
	Id        uuid.UUID `json:"id"`
	UnlockKey string    `json:"unlock_key"`
	Data      string    `json:"data"`
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	return &Store{db, path}, err
}

func (s *Store) Initialize() error {
	_, err := s.Exec("CREATE TABLE IF NOT EXISTS data (id VARCHAR(36), unlock_key TEXT, data TEXT, PRIMARY KEY (id, unlock_key))")
	return err
}

func (s *Store) InsertOrReplaceData(data Data) error {
	stmt, err := s.Prepare("INSERT OR REPLACE INTO data (id, unlock_key, data) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(data.Id.String(), data.UnlockKey, data.Data)
	return err
}

func (s *Store) SelectData(id uuid.UUID, unlockKey string) (*Data, error) {
	rows, err := s.Query("SELECT * FROM data WHERE id = ? AND unlock_key = ? LIMIT 1", id.String(), unlockKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		id_        string
		unlockKey_ string
		data       string
	)
	for rows.Next() {
		err := rows.Scan(&id_, &unlockKey_, &data)
		if err != nil {
			return nil, err
		}
		return &Data{id, unlockKey, data}, nil
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return nil, nil
}
