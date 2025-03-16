package repository

import (
	"database/sql"
	_ "fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"music-library/internal/models"
)

// PostgresRepository handles database operations for the music library
type PostgresRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPostgresRepository creates a new instance of PostgresRepository
func NewPostgresRepository(db *sqlx.DB, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}
}

// AddSong adds a new song to the database
func (r *PostgresRepository) AddSong(group, song, releaseDate, text, link string) (int, error) {
	r.logger.Debug("Adding song to database", zap.String("group", group), zap.String("song", song))
	query := `
		INSERT INTO songs (group_name, song_name, release_date, text, link, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING id`
	var id int
	err := r.db.QueryRow(query, group, song, releaseDate, text, link).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to add song", zap.Error(err))
		return 0, err
	}
	r.logger.Info("Song added to database", zap.Int("id", id))
	return id, nil
}

// GetSongs retrieves a list of songs with filtering and pagination
func (r *PostgresRepository) GetSongs(group, song string, page, limit int) ([]models.Song, error) {
	r.logger.Debug("Fetching songs from database", zap.String("group", group), zap.String("song", song))
	offset := (page - 1) * limit
	query := `SELECT * FROM songs WHERE group_name ILIKE $1 AND song_name ILIKE $2 
		ORDER BY id LIMIT $3 OFFSET $4`
	rows, err := r.db.Queryx(query, "%"+group+"%", "%"+song+"%", limit, offset)
	if err != nil {
		r.logger.Error("Failed to fetch songs", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var songs []models.Song
	for rows.Next() {
		var s models.Song
		err := rows.StructScan(&s)
		if err != nil {
			r.logger.Error("Failed to scan song", zap.Error(err))
			return nil, err
		}
		songs = append(songs, s)
	}

	r.logger.Info("Songs fetched from database", zap.Int("count", len(songs)))
	return songs, nil
}

// GetSongByID retrieves a song by its ID
func (r *PostgresRepository) GetSongByID(id int) (models.Song, error) {
	r.logger.Debug("Fetching song by ID", zap.Int("id", id))
	var song models.Song
	err := r.db.Get(&song, "SELECT * FROM songs WHERE id = $1", id)
	if err != nil {
		r.logger.Error("Failed to fetch song", zap.Int("id", id), zap.Error(err))
		return song, err
	}
	r.logger.Info("Song fetched from database", zap.Int("id", id))
	return song, nil
}

// UpdateSong updates an existing song in the database
func (r *PostgresRepository) UpdateSong(id int, group, song, releaseDate, text, link string) error {
	r.logger.Debug("Updating song in database", zap.Int("id", id))
	query := `UPDATE songs SET group_name = $2, song_name = $3, release_date = $4, text = $5, link = $6, updated_at = NOW() 
		WHERE id = $1`
	result, err := r.db.Exec(query, id, group, song, releaseDate, text, link)
	if err != nil {
		r.logger.Error("Failed to update song", zap.Int("id", id), zap.Error(err))
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to check rows affected", zap.Int("id", id), zap.Error(err))
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	r.logger.Info("Song updated in database", zap.Int("id", id))
	return nil
}

// DeleteSong deletes a song from the database
func (r *PostgresRepository) DeleteSong(id int) error {
	r.logger.Debug("Deleting song from database", zap.Int("id", id))
	query := "DELETE FROM songs WHERE id = $1"
	result, err := r.db.Exec(query, id)
	if err != nil {
		r.logger.Error("Failed to delete song", zap.Int("id", id), zap.Error(err))
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to check rows affected", zap.Int("id", id), zap.Error(err))
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	r.logger.Info("Song deleted from database", zap.Int("id", id))
	return nil
}

// TruncateSongs truncates the songs table and resets the ID sequence
func (r *PostgresRepository) TruncateSongs() error {
	r.logger.Debug("Truncating table")
	_, err := r.db.Exec("TRUNCATE TABLE songs RESTART IDENTITY")
	if err != nil {
		r.logger.Error("Failed to truncate table", zap.Error(err))
		return err
	}
	r.logger.Info("Table truncated in database")
	return nil
}
