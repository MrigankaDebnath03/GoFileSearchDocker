package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/go-chi/chi/v5"
	lru "github.com/hashicorp/golang-lru/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Product struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

var (
	db        *sql.DB
	index     bleve.Index
	cache     *lru.Cache[int, Product]
	cacheLock sync.RWMutex
)

func main() {
	initDB()
	defer db.Close()

	initCache()
	createSearchIndex()

	r := chi.NewRouter()
	r.Get("/search", searchHandler())
	r.Post("/products", addProductHandler())
	r.Delete("/products/{id}", deleteProductHandler())

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	log.Println("Server running on :8080")

	handleShutdown(srv)
}

func initDB() {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	var err error
	db, err = sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	for i := 0; i < 5; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func initCache() {
	cacheSize, err := strconv.Atoi(os.Getenv("CACHE_SIZE"))
	if err != nil || cacheSize <= 0 {
		cacheSize = 10000
	}

	var cacheErr error
	cache, cacheErr = lru.New[int, Product](cacheSize)
	if cacheErr != nil {
		log.Fatal("Failed to create cache:", cacheErr)
	}
}

func createSearchIndex() {
	mapping := bleve.NewIndexMapping()
	docMapping := bleve.NewDocumentMapping()

	nameField := bleve.NewTextFieldMapping()
	nameField.Analyzer = "en"
	docMapping.AddFieldMappingsAt("Name", nameField)

	mapping.AddDocumentMapping("product", docMapping)
	mapping.DefaultAnalyzer = "en"

	var err error
	index, err = bleve.NewMemOnly(mapping)
	if err != nil {
		log.Fatal("Failed to create search index:", err)
	}

	rows, err := db.Query("SELECT id, name FROM products")
	if err != nil {
		log.Fatal("Failed to load search index:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			log.Fatal("Scan error:", err)
		}
		index.Index(strconv.Itoa(p.ID), map[string]interface{}{
			"ID":   p.ID,
			"Name": p.Name,
		})
	}
	log.Println("Search index created")
}

func searchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing search query", http.StatusBadRequest)
			return
		}

		searchRequest := bleve.NewSearchRequest(bleve.NewMatchQuery(query))
		searchRequest.Size = 50
		searchResult, err := index.Search(searchRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		results := make([]Product, 0, 50)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, hit := range searchResult.Hits {
			wg.Add(1)
			go func(idStr string) {
				defer wg.Done()
				id, _ := strconv.Atoi(idStr)

				// Try cache first
				cacheLock.RLock()
				if product, ok := cache.Get(id); ok {
					cacheLock.RUnlock()
					mu.Lock()
					results = append(results, product)
					mu.Unlock()
					return
				}
				cacheLock.RUnlock()

				// Query database if not in cache
				var p Product
				err := db.QueryRow(
					"SELECT id, name, category FROM products WHERE id = $1",
					id,
				).Scan(&p.ID, &p.Name, &p.Category)
				if err != nil {
					return
				}

				// Add to cache
				cacheLock.Lock()
				cache.Add(p.ID, p)
				cacheLock.Unlock()

				mu.Lock()
				results = append(results, p)
				mu.Unlock()
			}(hit.ID)
		}

		wg.Wait()
		if len(results) > 50 {
			results = results[:50]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func addProductHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := db.QueryRow(
			"INSERT INTO products (name, category) VALUES ($1, $2) RETURNING id",
			p.Name, p.Category,
		).Scan(&p.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update search index
		index.Index(strconv.Itoa(p.ID), map[string]interface{}{
			"ID":   p.ID,
			"Name": p.Name,
		})

		// Add to cache
		cacheLock.Lock()
		cache.Add(p.ID, p)
		cacheLock.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	}
}

func deleteProductHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		_, err = db.Exec("DELETE FROM products WHERE id = $1", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Remove from search index
		index.Delete(strconv.Itoa(id))

		// Remove from cache
		cacheLock.Lock()
		cache.Remove(id)
		cacheLock.Unlock()

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleShutdown(srv *http.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Server stopped")
}
