package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"music-library/internal/models"
	"music-library/internal/repository"
	"music-library/internal/service"
)

// AddSongRequest defines the request body for adding a song
type AddSongRequest struct {
	Group string `json:"group" validate:"required"`
	Song  string `json:"song" validate:"required"`
}

// UpdateSongRequest defines the request body for updating a song
type UpdateSongRequest struct {
	Group       string `json:"group"`
	Song        string `json:"song"`
	ReleaseDate string `json:"release_date"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}

// ErrorResponse defines the structure of an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

func setupTest(t *testing.T) (*gin.Engine, *sqlx.DB, func()) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}

	// Устанавливаем EXTERNAL_API_URL для тестов (хотя в локальной среде он не будет использоваться)
	os.Setenv("EXTERNAL_API_URL", "http://mock-api:8081")

	db, err := sqlx.Connect("postgres", "host=localhost port=5432 user=postgres password=123456 dbname=music_library sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	repo := repository.NewPostgresRepository(db, logger)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	svc := service.NewMusicService(repo, logger, httpClient)
	handler := NewHandler(svc, logger)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/songs", handler.AddSong)
	r.GET("/songs", handler.GetSongs)
	r.GET("/songs/:id/verses", handler.GetVerses)
	r.PUT("/songs/:id", handler.UpdateSong)
	r.DELETE("/songs/:id", handler.DeleteSong)
	r.POST("/songs/truncate", handler.TruncateSongs)

	cleanup := func() {
		_, err := db.Exec("TRUNCATE TABLE songs RESTART IDENTITY")
		if err != nil {
			t.Logf("Failed to truncate table in cleanup: %v", err)
		}
		db.Close()
	}

	return r, db, cleanup
}

func TestAddSong(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	t.Run("Successful AddSong", func(t *testing.T) {
		reqBody := AddSongRequest{Group: "Muse", Song: "Supermassive Black Hole"}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/songs", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]int
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotZero(t, resp["id"])

		// Проверка в БД
		var song models.Song
		err = db.Get(&song, "SELECT * FROM songs WHERE id=$1", resp["id"])
		assert.NoError(t, err)
		assert.Equal(t, "Muse", song.Group)
	})

	t.Run("Invalid Request Body", func(t *testing.T) {
		reqBody := AddSongRequest{Group: "", Song: ""}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/songs", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, "Field validation")
	})
}

func TestGetSongs(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	// Подготовка данных
	_, err := db.Exec(`INSERT INTO songs (group_name, song_name, release_date, text, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		"Muse", "Supermassive Black Hole", "16.07.2006", "Verse 1\n\nVerse 2", "https://example.com")
	assert.NoError(t, err)

	t.Run("Successful GetSongs", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/songs?group=Muse&page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var songs []models.Song
		err := json.Unmarshal(w.Body.Bytes(), &songs)
		assert.NoError(t, err)
		assert.Len(t, songs, 1)
		assert.Equal(t, "Muse", songs[0].Group)
	})

	t.Run("Invalid Page", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/songs?page=invalid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid page number", resp.Error)
	})
}

func TestGetVerses(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	// Подготовка данных
	var songID int
	err := db.QueryRow(`INSERT INTO songs (group_name, song_name, release_date, text, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`,
		"Muse", "Supermassive Black Hole", "16.07.2006", "Verse 1\n\nVerse 2\n\nVerse 3", "https://example.com").Scan(&songID)
	assert.NoError(t, err)

	t.Run("Successful GetVerses", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/songs/%d/verses?page=1&limit=2", songID), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var verses []service.Verse
		err := json.Unmarshal(w.Body.Bytes(), &verses)
		assert.NoError(t, err)
		assert.Len(t, verses, 2)
		assert.Equal(t, "Verse 1", verses[0].Text)
		assert.Equal(t, "Verse 2", verses[1].Text)
	})

	t.Run("Song Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/songs/999/verses", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Song not found", resp.Error)
	})
}

func TestUpdateSong(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	// Подготовка данных
	var songID int
	err := db.QueryRow(`INSERT INTO songs (group_name, song_name, release_date, text, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`,
		"Muse", "Supermassive Black Hole", "16.07.2006", "Verse 1", "https://example.com").Scan(&songID)
	assert.NoError(t, err)

	t.Run("Successful UpdateSong", func(t *testing.T) {
		reqBody := UpdateSongRequest{
			Group:       "Muse",
			Song:        "New Song",
			ReleaseDate: "01.01.2007",
			Text:        "Updated Verse 1",
			Link:        "https://newlink.com",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/songs/%d", songID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Song updated successfully", resp["message"]) // Updated to match handler.go

		// Проверка обновления в БД
		var song models.Song
		err = db.Get(&song, "SELECT * FROM songs WHERE id=$1", songID)
		assert.NoError(t, err)
		assert.Equal(t, "New Song", song.Song)
	})

	t.Run("Song Not Found", func(t *testing.T) {
		reqBody := UpdateSongRequest{Group: "Muse", Song: "New Song"}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/songs/999", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Song not found", resp.Error)
	})
}

func TestDeleteSong(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	// Подготовка данных
	var songID int
	err := db.QueryRow(`INSERT INTO songs (group_name, song_name, release_date, text, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`,
		"Muse", "Supermassive Black Hole", "16.07.2006", "Verse 1", "https://example.com").Scan(&songID)
	assert.NoError(t, err)

	t.Run("Successful DeleteSong", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/songs/%d", songID), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Song deleted successfully", resp["message"]) // Updated to match handler.go

		// Проверка удаления из БД
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM songs WHERE id=$1", songID)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Song Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/songs/999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Song not found", resp.Error)
	})
}

func TestTruncateSongs(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	// Подготовка данных
	_, err := db.Exec(`INSERT INTO songs (group_name, song_name, release_date, text, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		"Muse", "Supermassive Black Hole", "16.07.2006", "Verse 1", "https://example.com")
	assert.NoError(t, err)

	t.Run("Successful TruncateSongs", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/songs/truncate", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Table truncated and sequence reset", resp["message"]) // Updated to match handler.go

		// Проверка очистки таблицы
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM songs")
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestFullWorkflow(t *testing.T) {
	r, db, cleanup := setupTest(t)
	defer cleanup()

	// 1. Добавление песни
	reqBody := AddSongRequest{Group: "Muse", Song: "Supermassive Black Hole"}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/songs", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]int
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	songID := resp["id"]

	// Проверка в БД
	var song models.Song
	err = db.Get(&song, "SELECT * FROM songs WHERE id=$1", songID)
	assert.NoError(t, err)
	assert.Equal(t, "Muse", song.Group)

	// 2. Получение списка песен
	req, _ = http.NewRequest(http.MethodGet, "/songs?group=Muse", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var songs []models.Song
	err = json.Unmarshal(w.Body.Bytes(), &songs)
	assert.NoError(t, err)
	assert.Len(t, songs, 1)

	// 3. Получение куплетов
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/songs/%d/verses?page=1&limit=2", songID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var verses []service.Verse
	err = json.Unmarshal(w.Body.Bytes(), &verses)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(verses), 1)

	// 4. Обновление песни
	updateBody := UpdateSongRequest{Group: "Muse", Song: "New Song", ReleaseDate: "01.01.2007"}
	updateBytes, _ := json.Marshal(updateBody)
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/songs/%d", songID), bytes.NewBuffer(updateBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверка обновления в БД
	err = db.Get(&song, "SELECT * FROM songs WHERE id=$1", songID)
	assert.NoError(t, err)
	assert.Equal(t, "New Song", song.Song)

	// 5. Удаление песни
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/songs/%d", songID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверка удаления
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM songs WHERE id=$1", songID)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// Проверка полного списка
	req, _ = http.NewRequest(http.MethodGet, "/songs?group=Muse", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	err = json.Unmarshal(w.Body.Bytes(), &songs)
	assert.NoError(t, err)
	assert.Len(t, songs, 0)
}
