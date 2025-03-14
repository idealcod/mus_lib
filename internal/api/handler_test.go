package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"music-library/internal/models"
	_ "music-library/internal/repository"
	"music-library/internal/service"
)

type MockHTTPClient struct {
	GetFunc func(url string) (*http.Response, error)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.GetFunc(url)
}

type MockRepository struct {
	AddSongFunc       func(group, song, releaseDate, text, link string) (int, error)
	GetSongsFunc      func(group, song, releaseDate string, limit, offset int) ([]models.Song, error)
	GetVersesFunc     func(songID int) (string, error)
	UpdateSongFunc    func(id int, group, song, releaseDate, text, link string) error
	DeleteSongFunc    func(id int) error
	TruncateTableFunc func() error
}

func (m *MockRepository) AddSong(group, song, releaseDate, text, link string) (int, error) {
	return m.AddSongFunc(group, song, releaseDate, text, link)
}

func (m *MockRepository) GetSongs(group, song, releaseDate string, limit, offset int) ([]models.Song, error) {
	return m.GetSongsFunc(group, song, releaseDate, limit, offset)
}

func (m *MockRepository) GetVerses(songID int) (string, error) {
	return m.GetVersesFunc(songID)
}

func (m *MockRepository) UpdateSong(id int, group, song, releaseDate, text, link string) error {
	return m.UpdateSongFunc(id, group, song, releaseDate, text, link)
}

func (m *MockRepository) DeleteSong(id int) error {
	return m.DeleteSongFunc(id)
}

func (m *MockRepository) TruncateTable() error {
	return m.TruncateTableFunc()
}

func TestHandler_AddSong(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, errors.New("external API unavailable")
		},
	}

	mockRepo := &MockRepository{
		AddSongFunc: func(group, song, releaseDate, text, link string) (int, error) {
			return 1, nil
		},
		GetSongsFunc: func(group, song, releaseDate string, limit, offset int) ([]models.Song, error) {
			return nil, nil
		},
		GetVersesFunc: func(songID int) (string, error) {
			return "", nil
		},
		UpdateSongFunc: func(id int, group, song, releaseDate, text, link string) error {
			return nil
		},
		DeleteSongFunc: func(id int) error {
			return nil
		},
		TruncateTableFunc: func() error {
			return nil
		},
	}

	svc := service.NewMusicService(mockRepo, logger, mockClient)
	handler := NewHandler(svc, logger)

	r := gin.Default()
	r.POST("/songs", handler.AddSong)

	body := AddSongRequest{Group: "Muse", Song: "Supermassive Black Hole"}
	bodyBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/songs", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]int
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 1, response["id"])
}

func TestHandler_AddSong_InvalidBody(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, errors.New("external API unavailable")
		},
	}

	mockRepo := &MockRepository{
		AddSongFunc: func(group, song, releaseDate, text, link string) (int, error) {
			return 0, nil
		},
		GetSongsFunc: func(group, song, releaseDate string, limit, offset int) ([]models.Song, error) {
			return nil, nil
		},
		GetVersesFunc: func(songID int) (string, error) {
			return "", nil
		},
		UpdateSongFunc: func(id int, group, song, releaseDate, text, link string) error {
			return nil
		},
		DeleteSongFunc: func(id int) error {
			return nil
		},
		TruncateTableFunc: func() error {
			return nil
		},
	}

	svc := service.NewMusicService(mockRepo, logger, mockClient)
	handler := NewHandler(svc, logger)

	r := gin.Default()
	r.POST("/songs", handler.AddSong)

	body := []byte(`{}`)
	req, _ := http.NewRequest("POST", "/songs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "required")
}

func TestHandler_DeleteSong(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, errors.New("external API unavailable")
		},
	}

	mockRepo := &MockRepository{
		AddSongFunc: func(group, song, releaseDate, text, link string) (int, error) {
			return 0, nil
		},
		GetSongsFunc: func(group, song, releaseDate string, limit, offset int) ([]models.Song, error) {
			return nil, nil
		},
		GetVersesFunc: func(songID int) (string, error) {
			return "", nil
		},
		UpdateSongFunc: func(id int, group, song, releaseDate, text, link string) error {
			return nil
		},
		DeleteSongFunc: func(id int) error {
			if id == 1 {
				return nil
			}
			return sql.ErrNoRows
		},
		TruncateTableFunc: func() error {
			return nil
		},
	}

	svc := service.NewMusicService(mockRepo, logger, mockClient)
	handler := NewHandler(svc, logger)

	r := gin.Default()
	r.DELETE("/songs/:id", handler.DeleteSong)

	req, _ := http.NewRequest("DELETE", "/songs/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Song deleted", response["message"])

	req, _ = http.NewRequest("DELETE", "/songs/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Song not found", response["error"])
}
