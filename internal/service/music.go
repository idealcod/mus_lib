package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	_ "github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"music-library/internal/models"
	"music-library/internal/repository"
)

// Verse represents a single verse of a song
type Verse struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

// MusicService handles the business logic for music operations
type MusicService struct {
	repo       *repository.PostgresRepository
	logger     *zap.Logger
	httpClient *http.Client
}

// NewMusicService creates a new instance of MusicService
func NewMusicService(repo *repository.PostgresRepository, logger *zap.Logger, httpClient *http.Client) *MusicService {
	return &MusicService{
		repo:       repo,
		logger:     logger,
		httpClient: httpClient,
	}
}

// AddSong adds a new song to the database, fetching additional data from an external API if available
func (s *MusicService) AddSong(group, song string) (int, error) {
	s.logger.Info("Adding song", zap.String("group", group), zap.String("song", song))

	releaseDate, text, link := s.fetchExternalData(group, song)
	if releaseDate == "" || text == "" || link == "" {
		s.logger.Warn("External API unavailable, using mock data", zap.Error(nil))
		releaseDate = "01.01.2000"
		text = "Verse 1\n\nVerse 2\n\nVerse 3"
		link = "https://example.com"
	}

	id, err := s.repo.AddSong(group, song, releaseDate, text, link)
	if err != nil {
		s.logger.Error("Failed to add song to database", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// fetchExternalData fetches song details from an external API
func (s *MusicService) fetchExternalData(group, song string) (releaseDate, text, link string) {
	apiURL := os.Getenv("EXTERNAL_API_URL")
	if apiURL == "" {
		s.logger.Error("EXTERNAL_API_URL environment variable not set")
		return "", "", ""
	}

	s.logger.Debug("Using EXTERNAL_API_URL", zap.String("api_url", apiURL))
	url := fmt.Sprintf("%s/info?group=%s&song=%s", apiURL, url.QueryEscape(group), url.QueryEscape(song))
	s.logger.Debug("Fetching data from external API", zap.String("url", url))

	resp, err := s.httpClient.Get(url)
	if err != nil {
		s.logger.Warn("Failed to fetch data from external API", zap.Error(err))
		return "", "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("External API returned non-OK status", zap.Int("status_code", resp.StatusCode))
		return "", "", ""
	}

	var data struct {
		ReleaseDate string `json:"release_date"`
		Text        string `json:"text"`
		Link        string `json:"link"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		s.logger.Warn("Failed to decode external API response", zap.Error(err))
		return "", "", ""
	}

	return data.ReleaseDate, data.Text, data.Link
}

// GetSongs retrieves a list of songs with filtering and pagination
func (s *MusicService) GetSongs(group, song string, page, limit int) ([]models.Song, error) {
	s.logger.Debug("Fetching songs", zap.String("group", group), zap.String("song", song))
	songs, err := s.repo.GetSongs(group, song, page, limit)
	if err != nil {
		s.logger.Error("Failed to fetch songs from database", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Songs fetched successfully", zap.Int("count", len(songs)))
	return songs, nil
}

// GetVerses retrieves verses for a song with pagination
func (s *MusicService) GetVerses(songID int, page, limit int) ([]Verse, error) {
	s.logger.Debug("Fetching verses for song", zap.Int("song_id", songID))
	song, err := s.repo.GetSongByID(songID)
	if err != nil {
		s.logger.Error("Failed to fetch song", zap.Int("song_id", songID), zap.Error(err))
		return nil, err
	}

	// Split text into verses by "\n\n"
	verses := strings.Split(song.Text, "\n\n")
	totalVerses := len(verses)
	start := (page - 1) * limit
	end := start + limit
	if start >= totalVerses {
		return []Verse{}, nil
	}
	if end > totalVerses {
		end = totalVerses
	}

	result := make([]Verse, 0, end-start)
	for i := start; i < end; i++ {
		verseText := strings.TrimSpace(verses[i])
		result = append(result, Verse{Number: i + 1, Text: verseText})
	}

	s.logger.Info("Verses retrieved successfully", zap.Int("song_id", songID), zap.Int("total_verses", totalVerses))
	return result, nil
}

// UpdateSong updates an existing song in the database
func (s *MusicService) UpdateSong(id int, group, song, releaseDate, text, link string) error {
	s.logger.Debug("Updating song", zap.Int("id", id))
	err := s.repo.UpdateSong(id, group, song, releaseDate, text, link)
	if err != nil {
		s.logger.Error("Failed to update song", zap.Int("id", id), zap.Error(err))
		return err
	}
	s.logger.Info("Song updated successfully", zap.Int("id", id))
	return nil
}

// DeleteSong deletes a song from the database
func (s *MusicService) DeleteSong(id int) error {
	s.logger.Debug("Deleting song", zap.Int("id", id))
	err := s.repo.DeleteSong(id)
	if err != nil {
		s.logger.Error("Failed to delete song", zap.Int("id", id), zap.Error(err))
		return err
	}
	s.logger.Info("Song deleted successfully", zap.Int("id", id))
	return nil
}

// TruncateSongs truncates the songs table and resets the ID sequence
func (s *MusicService) TruncateSongs() error {
	s.logger.Debug("Truncating table")
	err := s.repo.TruncateSongs()
	if err != nil {
		s.logger.Error("Failed to truncate table", zap.Error(err))
		return err
	}
	s.logger.Info("Table truncated successfully")
	return nil
}
