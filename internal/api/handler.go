package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	_ "music-library/internal/models"
	"music-library/internal/service"
)

// Handler handles HTTP requests for the music library API
type Handler struct {
	svc      *service.MusicService
	logger   *zap.Logger
	validate *validator.Validate
}

// NewHandler creates a new instance of Handler
func NewHandler(svc *service.MusicService, logger *zap.Logger) *Handler {
	return &Handler{
		svc:      svc,
		logger:   logger,
		validate: validator.New(),
	}
}

// AddSong handles the request to add a new song
func (h *Handler) AddSong(c *gin.Context) {
	h.logger.Info("Handling AddSong request")

	var req struct {
		Group string `json:"group" validate:"required"`
		Song  string `json:"song" validate:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the request
	if err := h.validate.Struct(req); err != nil {
		h.logger.Warn("Validation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field validation failed: " + err.Error()})
		return
	}

	h.logger.Debug("Request parsed", zap.String("group", req.Group), zap.String("song", req.Song))
	id, err := h.svc.AddSong(req.Group, req.Song)
	if err != nil {
		h.logger.Error("Failed to add song", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

// GetSongs handles the request to retrieve songs with filtering and pagination
func (h *Handler) GetSongs(c *gin.Context) {
	h.logger.Info("Handling GetSongs request")

	group := c.Query("group")
	song := c.Query("song")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		h.logger.Error("Invalid page number", zap.String("page", pageStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		h.logger.Error("Invalid limit", zap.String("limit", limitStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	songs, err := h.svc.GetSongs(group, song, page, limit)
	if err != nil {
		h.logger.Error("Failed to fetch songs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.logger.Info("Songs retrieved successfully", zap.Int("count", len(songs)))
	c.JSON(http.StatusOK, songs)
}

// GetVerses handles the request to retrieve verses for a song
func (h *Handler) GetVerses(c *gin.Context) {
	h.logger.Info("Handling GetVerses request")

	songIDStr := c.Param("id")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		h.logger.Error("Invalid song ID", zap.String("song_id", songIDStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		h.logger.Error("Invalid page number", zap.String("page", pageStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		h.logger.Error("Invalid limit", zap.String("limit", limitStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	verses, err := h.svc.GetVerses(songID, page, limit)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("Song not found", zap.Int("song_id", songID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
			return
		}
		h.logger.Error("Failed to fetch verses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.logger.Info("Verses retrieved successfully", zap.Int("song_id", songID), zap.Int("count", len(verses)))
	c.JSON(http.StatusOK, verses)
}

// UpdateSong handles the request to update an existing song
func (h *Handler) UpdateSong(c *gin.Context) {
	h.logger.Info("Handling UpdateSong request")

	songIDStr := c.Param("id")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		h.logger.Error("Invalid song ID", zap.String("song_id", songIDStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	var req struct {
		Group       string `json:"group"`
		Song        string `json:"song"`
		ReleaseDate string `json:"release_date"`
		Text        string `json:"text"`
		Link        string `json:"link"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Debug("Request parsed", zap.String("group", req.Group), zap.String("song", req.Song))
	err = h.svc.UpdateSong(songID, req.Group, req.Song, req.ReleaseDate, req.Text, req.Link)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("Song not found", zap.Int("song_id", songID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
			return
		}
		h.logger.Error("Failed to update song", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.logger.Info("Song updated successfully", zap.Int("song_id", songID))
	c.JSON(http.StatusOK, gin.H{"message": "Song updated successfully"})
}

// DeleteSong handles the request to delete a song
func (h *Handler) DeleteSong(c *gin.Context) {
	h.logger.Info("Handling DeleteSong request")

	songIDStr := c.Param("id")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		h.logger.Error("Invalid song ID", zap.String("song_id", songIDStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	err = h.svc.DeleteSong(songID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("Song not found", zap.Int("song_id", songID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
			return
		}
		h.logger.Error("Failed to delete song", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.logger.Info("Song deleted successfully", zap.Int("song_id", songID))
	c.JSON(http.StatusOK, gin.H{"message": "Song deleted successfully"})
}

// TruncateSongs handles the request to truncate the songs table
func (h *Handler) TruncateSongs(c *gin.Context) {
	h.logger.Info("Handling TruncateSongs request")

	err := h.svc.TruncateSongs()
	if err != nil {
		h.logger.Error("Failed to truncate table", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.logger.Info("Table truncated and sequence reset")
	c.JSON(http.StatusOK, gin.H{"message": "Table truncated and sequence reset"})
}
