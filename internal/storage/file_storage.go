package storage

import (
		"sync"
		"encoding/json"
		"os"
		"bufio"
)

type URLMapping struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileStorage struct {
		mu          sync.RWMutex
		filePath    string
		urlMappings map[string]string
}

func NewFileStorage(filePath string) (*FileStorage, error) {
		fs := &FileStorage{
			  filePath:    filePath,
				urlMappings: make(map[string]string),
		}

		err := fs.load()

		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		return fs, nil
}

func (fs *FileStorage) load() error {
		fs.mu.Lock()
		defer fs.mu.Unlock()

		file, err := os.Open(fs.filePath)

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

			fs.urlMappings[m.ShortURL] = m.OriginalURL
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		return nil
}

func (fs *FileStorage) save() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, err := os.OpenFile(fs.filePath, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)

  uuid := 1

  for short, original := range fs.urlMappings {
		m := URLMapping{UUID: uuid, ShortURL: short, OriginalURL: original}

		if err := encoder.Encode(m); err != nil {
			return err
		}

		uuid++
	}

	return nil
}

func (fs *FileStorage) Set(key, value string) error {
	  fs.mu.Lock()
	  fs.urlMappings[key] = value
	  fs.mu.Unlock()

	  return fs.save()
}

func (fs *FileStorage) Get(key string) (string, bool) {
	  fs.mu.RLock()
	  defer fs.mu.RUnlock()
	  value, found := fs.urlMappings[key]

	  return value, found
}
