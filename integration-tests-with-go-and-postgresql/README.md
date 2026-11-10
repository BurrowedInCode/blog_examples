# Integration Tests with Go and PostgreSQL

A runnable companion example demonstrating a unique schema per test, SQL migrations, automatic cleanup, and a book insert-and-fetch round trip.

## Requirements

- Go 1.26 or newer.
- Docker with Docker Compose and a running Docker daemon.
- Local port 55439 available.

## Files

```text
compose.yaml
db/migrations/001_create_books.up.sql
internal/store/store.go
internal/store/store_test.go
go.mod
go.sum
```

`store.go` defines the connection-pool wrapper. `store_test.go` contains the complete helper and round-trip test. The migration path is relative to `internal/store`, the test package's working directory, so `../../db/migrations` resolves to this example's migrations directory.

## Run the example

Run these commands from this directory:

```sh
cd integration-tests-with-go-and-postgresql
go mod download
docker compose up -d --wait
export TEST_DATABASE_URL='postgres://test:test@127.0.0.1:55439/books_test?sslmode=disable'
go test -v -count=1 -timeout=60s ./...
```

Look for `--- PASS: TestBookRoundTrip`, not `SKIP`. Without `TEST_DATABASE_URL`, the helper skips the test. `-count=1` disables cached test results.

Compose starts a disposable PostgreSQL 17 container, binds it only to localhost, and stores database files in temporary memory-backed storage. The example credentials and disabled SSL are for local testing only. **Never use development or production credentials.** Stopping the container discards its data.

## What the test does

1. Creates a uniquely named schema.
2. Configures every pool connection to use that schema through `search_path`.
3. Reads and executes the migration to create `books`.
4. Inserts a book, captures its generated ID, and fetches it again.
5. Compares the retrieved title and author with the original values.
6. Drops the schema and closes the pool through `t.Cleanup`.

Schemas isolate database objects, not PostgreSQL instances or permissions. This example tests SQL persistence directly, rather than a separate application's storage methods.

## Check cleanup

After the test, this should return `0`:

```sh
docker compose exec -T postgres psql -U test -d books_test -Atc "SELECT count(*) FROM information_schema.schemata WHERE schema_name ~ '^app_test_[0-9a-f]{32}$';"
```

## Shut down

```sh
docker compose down --volumes
unset TEST_DATABASE_URL
```

Run the shutdown command even if a test fails. Test cleanup removes the schema, but does not stop the container.
