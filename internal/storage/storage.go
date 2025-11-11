package storage

import (
		"sync"
		"encoding/json"
		"os"
		"bufio"
		"context"

		"github.com/flash1nho/go-musthave-shortener-tpl/internal/db"
)

type URLMapping struct {
	  UUID        int    `json:"uuid"`
	  ShortURL    string `json:"short_url"`
	  OriginalURL string `json:"original_url"`
}

type Storage struct {
		mu          sync.RWMutex
		DatabaseDSN string
		filePath    string
		urlMappings map[string]string
}

func NewStorage(databaseDSN string, filePath string) (*Storage, error) {
		s := &Storage{
			  DatabaseDSN: databaseDSN,
			  filePath:    filePath,
				urlMappings: make(map[string]string),
		}

    if s.DatabaseDSN != "" {
				err := s.dbLoad()

				if err != nil && !os.IsNotExist(err) {
						return nil, err
				}
		} else if s.filePath != "" {
				err := s.fileLoad()

				if err != nil {
						return nil, err
				}
		}

		return s, nil
}

func (s *Storage) fileLoad() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_RDONLY, 0644)

		if err != nil {
				return err
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Bytes()

			var m URLMapping

			if err := json.Unmarshal(line, &m); err != nil {
					continue
			}

			s.urlMappings[m.ShortURL] = m.OriginalURL
		}

		if err := scanner.Err(); err != nil {
				return err
		}

		return nil
}

func (s *Storage) dbLoad() error {
		s.mu.Lock()
		defer s.mu.Unlock()

    conn, err := db.Connect(s.DatabaseDSN)

		if err != nil {
				return err
		}

		defer conn.Close(context.Background())

		rows, err := conn.Query(context.Background(), "SELECT original_url, short_url FROM shorten_urls;")

		if err != nil {
		    return err
		}

		for rows.Next() {
		    var (
		        originalURL string
		        shortURL    string
		    )

		    err = rows.Scan(&originalURL, &shortURL)

		    if err != nil {
		        return err
		    }

		    s.urlMappings[shortURL] = originalURL
		}

		return nil
}

func (s *Storage) save(key string, value string) error {
		s.mu.Lock()
		defer s.mu.Unlock()

		var err error = nil

	  if s.DatabaseDSN != "" {
				err = s.dbSave(value, key)
		} else if s.filePath != "" {
				err = s.fileSave()
		}

		if err != nil {
				return err
		}

		return nil
}

func (s *Storage) fileSave() error {
		file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
				return err
		}

		defer file.Close()

		encoder := json.NewEncoder(file)

	  uuid := 1

	  for short, original := range s.urlMappings {
			m := URLMapping{UUID: uuid, ShortURL: short, OriginalURL: original}

			if err := encoder.Encode(m); err != nil {
					return err
			}

			uuid++
		}

		return nil
}

func (s *Storage) dbSave(originalURL string, shortURL string) error {
    conn, err := db.Connect(s.DatabaseDSN)

		if err != nil {
				return err
		}

		defer conn.Close(context.Background())

    insertSQL := `INSERT INTO shorten_urls (original_url, short_url) VALUES ($1, $2)`
    _, err = conn.Exec(context.Background(), insertSQL, originalURL, shortURL)

		if err != nil {
				return err
		}

		return nil
}

func (s *Storage) Set(key string, value string) error {
	  s.mu.Lock()
	  s.urlMappings[key] = value
	  s.mu.Unlock()

	  return s.save(key, value)
}

func (s *Storage) Get(key string) (string, bool) {
	  s.mu.RLock()
	  defer s.mu.RUnlock()
	  value, found := s.urlMappings[key]

	  return value, found
}
