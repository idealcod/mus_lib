package repository

import "music-library/internal/models"

type Repository interface {
	AddSong(group, song, releaseDate, text, link string) (int, error)
	GetSongs(group, song, releaseDate, text, link, createdAt, updatedAt string, limit, offset int) ([]models.Song, error)
	UpdateSong(id int, group, song, releaseDate, text, link string) error
	DeleteSong(id int) error
	TruncateSongs() error
	GetSongByID(id int) (models.Song, error)
}
