package seeddata

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
)

// WriteFile writes books as gzip-compressed JSON Lines.
func WriteFile(path string, books []Book) error {

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(gz)
	for _, b := range books {
		if err := enc.Encode(b); err != nil {
			return err
		}
	}

	if err := gz.Close(); err != nil {
		return err
	}
	return f.Close()
}

// ReadFile reads books written by WriteFile.
func ReadFile(path string) ([]Book, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var books []Book
	dec := json.NewDecoder(bufio.NewReader(gz))
	for dec.More() {
		var b Book
		if err := dec.Decode(&b); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, nil
}
