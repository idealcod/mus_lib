package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"music-library/internal/models"
	"music-library/internal/repository"
)

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

type MusicService struct {
	repo   repository.Repository
	logger *zap.Logger
	client HTTPClient
}

func NewMusicService(repo repository.Repository, logger *zap.Logger, client HTTPClient) *MusicService {
	return &MusicService{
		repo:   repo,
		logger: logger,
		client: client,
	}
}

func (s *MusicService) AddSong(group, song string) (int, error) {
	url := "http://external-api:8080/info?group=" + group + "&song=" + song
	s.logger.Info("Sending request to external API", zap.String("url", url))
	s.logger.Debug("Executing HTTP GET request")

	resp, err := s.client.Get(url)
	if err != nil {
		s.logger.Warn("External API unavailable, using mock data", zap.Error(err))
		return s.addSongWithMockData(group, song)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("External API returned non-200 status", zap.Int("status_code", resp.StatusCode))
		return s.addSongWithMockData(group, song)
	}

	var songDetail struct {
		ReleaseDate string `json:"releaseDate"`
		Text        string `json:"text"`
		Link        string `json:"link"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read API response", zap.Error(err))
		return s.addSongWithMockData(group, song)
	}

	if err := json.Unmarshal(body, &songDetail); err != nil {
		s.logger.Error("Failed to unmarshal API response", zap.Error(err))
		return s.addSongWithMockData(group, song)
	}

	s.logger.Info("Successfully fetched data from external API",
		zap.String("releaseDate", songDetail.ReleaseDate),
		zap.String("text", songDetail.Text),
		zap.String("link", songDetail.Link))

	id, err := s.repo.AddSong(group, song, songDetail.ReleaseDate, songDetail.Text, songDetail.Link)
	if err != nil {
		s.logger.Error("Failed to save song to repository", zap.Error(err))
		return 0, err
	}
	return id, nil
}

func (s *MusicService) addSongWithMockData(group, song string) (int, error) {
	s.logger.Debug("Using mock data for song")
	mockData := models.Song{
		GroupName:   group,
		SongName:    song,
		ReleaseDate: "2006-06-19",
		Text:        "Ooh baby, don't you know I suffer?\nOoh baby, can you hear me moan?",
		Link:        "https://example.com/muse/supermassive-black-hole",
	}
	s.logger.Info("Using mock data for song", zap.Any("mock_data", mockData))
	id, err := s.repo.AddSong(group, song, mockData.ReleaseDate, mockData.Text, mockData.Link)
	if err != nil {
		s.logger.Error("Failed to save mock song to repository", zap.Error(err))
		return 0, err
	}
	return id, nil
}

func (s *MusicService) GetSongs(group, song, releaseDate string, limit, offset int) ([]models.Song, error) {
	s.logger.Debug("Fetching songs", zap.String("group", group), zap.String("song", song), zap.String("releaseDate", releaseDate))
	songs, err := s.repo.GetSongs(group, song, releaseDate, limit, offset)
	if err != nil {
		s.logger.Error("Failed to fetch songs", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Songs fetched successfully", zap.Int("count", len(songs)))
	return songs, nil
}

func (s *MusicService) GetVerses(songID int, page, limit int) ([]string, error) {
	s.logger.Debug("Fetching verses for song", zap.Int("song_id", songID))
	text, err := s.repo.GetVerses(songID)
	if err != nil {
		s.logger.Error("Failed to fetch verses", zap.Error(err), zap.Int("song_id", songID))
		return nil, err
	}

	verses := splitVerses(text)
	total := len(verses)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []string{}, nil
	}
	if end > total {
		end = total
	}

	s.logger.Info("Verses retrieved successfully", zap.Int("song_id", songID), zap.Int("total_verses", total))
	return verses[start:end], nil
}

func (s *MusicService) UpdateSong(id int, group, song, releaseDate, text, link string) error {
	s.logger.Debug("Updating song", zap.Int("id", id))
	err := s.repo.UpdateSong(id, group, song, releaseDate, text, link)
	if err != nil {
		s.logger.Error("Failed to update song", zap.Error(err), zap.Int("id", id))
		return err
	}
	s.logger.Info("Song updated successfully", zap.Int("id", id))
	return nil
}

func (s *MusicService) DeleteSong(id int) error {
	s.logger.Debug("Deleting song", zap.Int("id", id))
	err := s.repo.DeleteSong(id)
	if err != nil {
		s.logger.Error("Failed to delete song", zap.Error(err), zap.Int("id", id))
		return err
	}
	s.logger.Info("Song deleted successfully", zap.Int("id", id))
	return nil
}

func (s *MusicService) TruncateSongs() error {
	s.logger.Debug("Truncating table")
	err := s.repo.TruncateTable()
	if err != nil {
		s.logger.Error("Failed to truncate table", zap.Error(err))
		return err
	}
	s.logger.Info("Table truncated successfully")
	return nil
}

func splitVerses(text string) []string {
	return strings.Split(text, "\n\n")
}
