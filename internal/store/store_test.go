package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	schemaName := "app_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schemaName}.Sanitize()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	ctx := context.Background()
	config.ConnConfig.RuntimeParams["search_path"] = quotedSchema

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("creating connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	_, err = pool.Exec(ctx, "CREATE SCHEMA "+quotedSchema)
	if err != nil {
		t.Fatalf("creating schema: %v", err)
	}
	t.Cleanup(func() {
		_, err := pool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		if err != nil {
			t.Errorf("dropping schema: %v", err)
		}
	})

	migrationFiles, err := filepath.Glob("../../db/migrations/*.up.sql")
	if err != nil {
		t.Fatalf("finding migrations: %v", err)
	}
	if len(migrationFiles) == 0 {
		t.Fatal("no migration files found")
	}

	for _, f := range migrationFiles {
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			t.Fatalf("applying migration %s: %v", f, err)
		}
	}
	return NewStore(pool)
}

func TestBookRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	type Book struct {
		Title  string
		Author string
	}

	want := Book{
		Title:  "The Hobbit",
		Author: "J. R. R. Tolkien",
	}

	// Insert a book and capture its generated ID.
	var id int64
	err := store.db.QueryRow(ctx, `
		INSERT INTO books (title, author)
		VALUES ($1, $2)
		RETURNING id
	`, want.Title, want.Author).Scan(&id)
	if err != nil {
		t.Fatalf("inserting book: %v", err)
	}

	// Retrieve the book using its ID.
	var got Book
	err = store.db.QueryRow(ctx, `
		SELECT title, author
		FROM books
		WHERE id = $1
	`, id).Scan(&got.Title, &got.Author)
	if err != nil {
		t.Fatalf("fetching book: %v", err)
	}

	// Verify that the stored values match the original input.
	if got != want {
		t.Errorf("book = %+v, want %+v", got, want)
	}
}
