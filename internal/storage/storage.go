package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

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

type URLDetails struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"originalURL"`
	IsDeleted   bool   `json:"is_deleted"`
	UserID      string `json:"user_id"`
}

type Storage struct {
	mu          sync.RWMutex
	filePath    string
	Pool        *pgxpool.Pool
	urlMappings map[string]URLDetails
}

type UpdateItem struct {
	UserID   string
	ShortURL string
}

type UpdateResult struct {
	ShortURL string
	Updated  bool
}

const numWorkers = 4

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
		urlMappings: make(map[string]URLDetails),
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

		s.urlMappings[m.ShortURL] = URLDetails{OriginalURL: m.OriginalURL, IsDeleted: false}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Storage) dbLoad() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.Pool.Query(context.TODO(), "SELECT original_url, short_url FROM shorten_urls WHERE is_deleted = FALSE")

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

		s.urlMappings[shortURL] = URLDetails{OriginalURL: originalURL, IsDeleted: false}
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

	for short, details := range s.urlMappings {
		m := URLMapping{UUID: uuid, ShortURL: short, OriginalURL: details.OriginalURL}

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
	if s.Pool == nil {
		return nil
	}

	pb := &pgx.Batch{}

	for shortURL, originalURL := range batch {
		pb.Queue(`INSERT INTO shorten_urls (original_url, short_url) VALUES ($1, $2)`, originalURL, shortURL)
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
	s.urlMappings[key] = URLDetails{OriginalURL: value, UserID: userID, IsDeleted: false}
	s.mu.Unlock()

	return s.save(key, value, userID)
}

func (s *Storage) SetBatch(batch map[string]string) error {
	s.mu.Lock()

	for shortURL, originalURL := range batch {
		s.urlMappings[shortURL] = URLDetails{OriginalURL: originalURL, IsDeleted: false}
	}

	s.mu.Unlock()

	return s.dbSaveBatch(batch)
}

func (s *Storage) Get(key string) (URLDetails, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, found := s.urlMappings[key]

	return value, found
}

func (s *Storage) GetURLsByUserID(userID string) ([]URLDetails, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var batch []URLDetails

	if s.Pool != nil {
		query := `SELECT original_url, short_url FROM shorten_urls WHERE user_id = $1 AND is_deleted = FALSE`
		rows, err := s.Pool.Query(context.TODO(), query, userID)

		if err != nil {
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var (
				originalURL string
				shortURL    string
			)

			err = rows.Scan(&originalURL, &shortURL)

			if err != nil {
				return nil, err
			}

			item := URLDetails{ShortURL: shortURL, OriginalURL: originalURL}
			batch = append(batch, item)
		}
	} else {
		for _, item := range s.urlMappings {
			if item.UserID == userID {
				batch = append(batch, item)
			}
		}
	}

	return batch, nil
}

func (s *Storage) DeleteBatch(userID string, ShortURLs []string) error {
	var items []UpdateItem

	for i := 0; i < len(ShortURLs); i++ {
		item := UpdateItem{
			UserID:   userID,
			ShortURL: ShortURLs[i],
		}

		items = append(items, item)
	}

	err := batchUpdateWithFanIn(s, context.TODO(), items)

	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) Close() error {
	var err error

	if s.Pool != nil {
		s.Pool.Close()
	} else if s.filePath != "" {
		err = s.fileSave()
	}

	if err != nil {
		return err
	}

	return nil
}

func batchUpdateWithFanIn(s *Storage, ctx context.Context, items []UpdateItem) error {
	jobs := make(chan []UpdateItem, numWorkers)

	results := make(chan UpdateResult, len(items))

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)

		go worker(ctx, s.Pool, jobs, results, &wg)
	}

	jobs <- items

	close(jobs)

	go func() {
		wg.Wait()

		close(results)
	}()

	for result := range results {
		if result.Updated {
			s.urlMappings[result.ShortURL] = URLDetails{IsDeleted: true}
		}
	}

	return nil
}

func worker(ctx context.Context, pool *pgxpool.Pool, jobs <-chan []UpdateItem, results chan<- UpdateResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for items := range jobs {
		batch := &pgx.Batch{}

		for _, item := range items {
			batch.Queue(`UPDATE shorten_urls SET is_deleted = TRUE WHERE user_id = $1 AND short_url = $2`, item.UserID, item.ShortURL)
		}

		br := pool.SendBatch(ctx, batch)

		for _, item := range items {
			_, err := br.Exec()

			if err != nil {
				results <- UpdateResult{ShortURL: item.ShortURL, Updated: false}
			} else {
				results <- UpdateResult{ShortURL: item.ShortURL, Updated: true}
			}
		}

		br.Close()
	}
}
