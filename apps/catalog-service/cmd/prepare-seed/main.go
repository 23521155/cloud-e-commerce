// Command prepare-seed cleans the scraped dataset into seed/books.jsonl.gz.
// The raw file is gitignored, so run this once locally and commit the output:
//
//	go run ./cmd/prepare-seed -in ../web/ref/antique_books.jsonl
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/fizzisme/catalog-service/internal/seeddata"
)

func main() {

	in := flag.String("in", "../web/ref/antique_books.jsonl", "raw JSON Lines dataset")
	out := flag.String("out", "seed/books.jsonl.gz", "cleaned seed file to write")
	flag.Parse()

	f, err := os.Open(*in)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var (
		books                      []seeddata.Book
		read                       int
		noPrice, notABook, badYear int
	)

	scanner := bufio.NewScanner(f)
	// Some lines carry long descriptions; raise the 64 KB default.
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		read++

		var raw seeddata.RawBook
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			log.Fatalf("line %d: %v", read, err)
		}

		b, err := seeddata.Clean(raw)
		switch {
		case errors.Is(err, seeddata.ErrNoPrice):
			noPrice++
		case errors.Is(err, seeddata.ErrNotABook):
			notABook++
		case errors.Is(err, seeddata.ErrBadYear):
			badYear++
		case err != nil:
			log.Fatalf("line %d: %v", read, err)
		default:
			books = append(books, b)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("seed", 0o755); err != nil {
		log.Fatal(err)
	}
	if err := seeddata.WriteFile(*out, books); err != nil {
		log.Fatal(err)
	}

	log.Printf("read %d, kept %d, skipped %d without price, %d non-books and %d with a bogus year → %s",
		read, len(books), noPrice, notABook, badYear, *out)
}
