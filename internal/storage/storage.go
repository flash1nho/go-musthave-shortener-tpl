package storage

import (
		"sync"
		"encoding/json"
		"os"
		"bufio"
		"context"
		"fmt"

		"github.com/flash1nho/go-musthave-shortener-tpl/internal/db"

    "github.com/jackc/pgx/v5"
		"github.com/jackc/pgx/v5/pgxpool"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

type URLMapping struct {
	  UUID        int    `json:"uuid"`
	  ShortURL    string `json:"short_url"`
	  OriginalURL string `json:"original_url"`
}

type Storage struct {
		mu          sync.RWMutex
		filePath    string
		Pool        *pgxpool.Pool
		urlMappings map[string]string
}

type Batch struct {
    urlMappings map[string]string
}

func NewStorage(filePath string, databaseDSN string) (*Storage, error) {
	  var pool *pgxpool.Pool = nil
	  var err error

    if databaseDSN != "" {
			  pool, err = db.Connect(databaseDSN)

			  if err != nil {
			  		return nil, err
			  }

		    m, err := migrate.New("file://migrations", databaseDSN)

		    if err != nil {
		    	  return nil, fmt.Errorf("ошибка загрузки миграций: %w", err)
		    }

		    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		        return nil, fmt.Errorf("ошибка запуска миграций: %w", err)
		    }
	  }

		s := &Storage{
			  filePath:    filePath,
			  Pool:        pool,
				urlMappings: make(map[string]string),
		}

    if s.Pool != nil {
				err = s.dbLoad()

				if err != nil && !os.IsNotExist(err) {
						return nil, err
				}
		} else if s.filePath != "" {
				err = s.fileLoad()

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

		rows, err := s.Pool.Query(context.TODO(), "SELECT original_url, short_url FROM shorten_urls;")

		if err != nil {
		    return err
		}

		defer rows.Close()

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

func (s *Storage) save(key string, value string, userID string) error {
		s.mu.Lock()
		defer s.mu.Unlock()

		var err error

	  if s.Pool != nil {
				err = s.dbSave(value, key, userID)
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

func (s *Storage) dbSave(originalURL string, shortURL string, userID string) error {
    insertSQL := `INSERT INTO shorten_urls (original_url, short_url, user_id) VALUES ($1, $2, $3)`
    _, err := s.Pool.Exec(context.TODO(), insertSQL, originalURL, shortURL, userID)

		if err != nil {
				return err
		}

		return nil
}

func (s *Storage) dbSaveBatch(batch map[string]string) error {
		pb := &pgx.Batch{}

    for shortURL, originalURL := range batch {
			  pb.Queue(`INSERT INTO shorten_urls (original_url, short_url) VALUES ($1, $2)`, originalURL, shortURL)
		}

		if s.Pool == nil {
				return nil
		}

		results := s.Pool.SendBatch(context.TODO(), pb)
		defer results.Close()

		for i := 0; i < len(batch); i++ {
			_, err := results.Exec()

			if err != nil {
				return err
			}
		}

		return nil
}

func (s *Storage) Set(key string, value string, userID string) error {
	  s.mu.Lock()
	  s.urlMappings[key] = value
	  s.mu.Unlock()

	  return s.save(key, value, userID)
}

func (s *Storage) SetBatch(batch map[string]string) error {
	  s.mu.Lock()

	  for shortURL, originalURL := range batch {
			  s.urlMappings[shortURL] = originalURL
		}

	  s.mu.Unlock()

	  return s.dbSaveBatch(batch)
}

func (s *Storage) Get(key string) (string, bool) {
	  s.mu.RLock()
	  defer s.mu.RUnlock()
	  value, found := s.urlMappings[key]

	  return value, found
}

func (s *Storage) GetURLsByUserID(userID string) (map[string]string, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

    query := `SELECT original_url, short_url FROM shorten_urls WHERE user_id = $1`
		rows, err := s.Pool.Query(context.TODO(), query, userID)

		if err != nil {
		    return nil, err
		}

		defer rows.Close()

    batch := &Batch{
        urlMappings: make(map[string]string),
    }

		for rows.Next() {
		    var (
		        originalURL string
		        shortURL    string
		    )

		    err = rows.Scan(&originalURL, &shortURL)

		    if err != nil {
		        return nil, err
		    }

		    batch.urlMappings[shortURL] = originalURL
		}

		return batch.urlMappings, nil
}
