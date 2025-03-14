# Music Library API

This is a RESTful API for managing a music library, built with Go, Gin, and PostgreSQL. It allows adding, retrieving, updating, and deleting songs, with support for pagination and verse extraction.

## Features
- Add a song with group and title (fetches details from an external API or uses mock data).
- Retrieve songs with filtering by group, song name, and release date.
- Get song verses with pagination.
- Update and delete songs.
- Truncate the songs table.
- 

## Requirements
- Go 1.21+
- PostgreSQL
- [Migrate CLI](https://github.com/golang-migrate/migrate) for database migrations
- External API (mocked in tests)

## Setup
1. Clone the repository: `git clone <repository-url>`
2. Install dependencies: `go mod download`
3. Set up PostgreSQL: `psql -U postgres -c "CREATE DATABASE music_library;"`
4. Set up .env file: (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, PORT)
5. Apply migrations: `migrate -path migrations -database "postgres://postgres:your_password@localhost:5432/music_library?sslmode=disable" up`
6. Run: `go run cmd/main.go`

## Testing
Run tests: `go test ./internal/api -v`