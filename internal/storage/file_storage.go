package storage

import (
		"sync"
		"encoding/json"
		"os"
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

		data, err := os.ReadFile(fs.filePath)

		if err != nil {
			return err
		}

		var mappings []URLMapping

		if err := json.Unmarshal(data, &mappings); err != nil {
			return err
		}

		for _, m := range mappings {
			fs.urlMappings[m.ShortURL] = m.OriginalURL
		}

		return nil
}

func (fs *FileStorage) save() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var mappings []URLMapping

	for short, original := range fs.urlMappings {
		uuid := len(mappings) + 1

		mappings = append(mappings, URLMapping{
			UUID:        uuid,
			ShortURL:    short,
			OriginalURL: original,
		})
	}

	data, err := json.MarshalIndent(mappings, "", "  ")

	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0644)
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
