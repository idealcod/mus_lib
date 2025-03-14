package repository

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"music-library/internal/models"
)

type Repository interface {
	AddSong(group, song, releaseDate, text, link string) (int, error)
	GetSongs(group, song, releaseDate string, limit, offset int) ([]models.Song, error)
	GetVerses(songID int) (string, error)
	UpdateSong(id int, group, song, releaseDate, text, link string) error
	DeleteSong(id int) error
	TruncateTable() error
}

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) AddSong(group, song, releaseDate, text, link string) (int, error) {
	var id int
	query := `INSERT INTO songs (group_name, song_name, release_date, text, link) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := r.db.QueryRow(query, group, song, releaseDate, text, link).Scan(&id)
	return id, err
}

func (r *PostgresRepository) GetSongs(group, song, releaseDate string, limit, offset int) ([]models.Song, error) {
	var songs []models.Song
	query := `SELECT id, group_name, song_name, release_date, text, link, created_at, updated_at 
	          FROM songs WHERE 1=1`
	args := []interface{}{}
	if group != "" {
		query += " AND group_name ILIKE $1"
		args = append(args, "%"+group+"%")
	}
	if song != "" {
		query += fmt.Sprintf(" AND song_name ILIKE $%d", len(args)+1)
		args = append(args, "%"+song+"%")
	}
	if releaseDate != "" {
		query += fmt.Sprintf(" AND release_date = $%d", len(args)+1)
		args = append(args, releaseDate)
	}
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	err := r.db.Select(&songs, query, args...)
	return songs, err
}

func (r *PostgresRepository) GetVerses(songID int) (string, error) {
	var text string
	query := "SELECT text FROM songs WHERE id = $1"
	err := r.db.QueryRow(query, songID).Scan(&text)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("song with id %d not found", songID)
	}
	return text, err
}

func (r *PostgresRepository) UpdateSong(id int, group, song, releaseDate, text, link string) error {
	query := `UPDATE songs 
	          SET group_name = $1, song_name = $2, release_date = $3, text = $4, link = $5, updated_at = CURRENT_TIMESTAMP 
	          WHERE id = $6`
	_, err := r.db.Exec(query, group, song, releaseDate, text, link, id)
	return err
}

func (r *PostgresRepository) DeleteSong(id int) error {
	query := "DELETE FROM songs WHERE id = $1"
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) TruncateTable() error {
	_, err := r.db.Exec("TRUNCATE TABLE songs RESTART IDENTITY")
	return err
}
