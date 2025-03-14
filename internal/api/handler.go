package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"music-library/internal/service"
)

type Handler struct {
	svc    *service.MusicService
	logger *zap.Logger
}

func NewHandler(svc *service.MusicService, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

type AddSongRequest struct {
	Group string `json:"group" binding:"required,min=1,max=255"`
	Song  string `json:"song" binding:"required,min=1,max=255"`
}

type UpdateSongRequest struct {
	Group       string `json:"group" binding:"required,min=1,max=255"`
	Song        string `json:"song" binding:"required,min=1,max=255"`
	ReleaseDate string `json:"release_date" binding:"max=10"`
	Text        string `json:"text"`
	Link        string `json:"link" binding:"max=255"`
}

func (h *Handler) AddSong(c *gin.Context) {
	h.logger.Info("Handling AddSong request")
	var req AddSongRequest
	h.logger.Debug("Parsing request body")
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.logger.Debug("Request parsed", zap.String("group", req.Group), zap.String("song", req.Song))

	id, err := h.svc.AddSong(req.Group, req.Song)
	if err != nil {
		h.logger.Error("Failed to add song", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	h.logger.Info("Song added successfully", zap.Int("id", id))
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *Handler) GetSongs(c *gin.Context) {
	h.logger.Info("Handling GetSongs request")
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

	group := c.Query("group")
	song := c.Query("song")
	releaseDate := c.Query("release_date")
	h.logger.Debug("Extracted query parameters", zap.String("group", group), zap.String("song", song), zap.String("release_date", releaseDate))

	songs, err := h.svc.GetSongs(group, song, releaseDate, limit, (page-1)*limit)
	if err != nil {
		h.logger.Error("Failed to get songs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	h.logger.Info("Songs retrieved successfully", zap.Int("count", len(songs)))
	c.JSON(http.StatusOK, songs)
}

func (h *Handler) GetVerses(c *gin.Context) {
	h.logger.Info("Handling GetVerses request")
	songIDStr := c.Param("id")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		h.logger.Error("Invalid song ID", zap.String("song_id", songIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "1")

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
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("Song not found", zap.Int("song_id", songID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
			return
		}
		h.logger.Error("Failed to get verses", zap.Error(err), zap.Int("song_id", songID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	h.logger.Info("Verses retrieved successfully", zap.Int("song_id", songID), zap.Int("count", len(verses)))
	c.JSON(http.StatusOK, verses)
}

func (h *Handler) UpdateSong(c *gin.Context) {
	h.logger.Info("Handling UpdateSong request")
	songIDStr := c.Param("id")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		h.logger.Error("Invalid song ID", zap.String("song_id", songIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	var req UpdateSongRequest
	h.logger.Debug("Parsing request body")
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.logger.Debug("Request parsed", zap.String("group", req.Group), zap.String("song", req.Song))

	err = h.svc.UpdateSong(songID, req.Group, req.Song, req.ReleaseDate, req.Text, req.Link)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("Song not found", zap.Int("song_id", songID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
			return
		}
		h.logger.Error("Failed to update song", zap.Error(err), zap.Int("song_id", songID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	h.logger.Info("Song updated successfully", zap.Int("song_id", songID))
	c.JSON(http.StatusOK, gin.H{"message": "Song updated"})
}

func (h *Handler) DeleteSong(c *gin.Context) {
	h.logger.Info("Handling DeleteSong request")
	songIDStr := c.Param("id")
	h.logger.Debug("Extracting song ID from URL", zap.String("song_id", songIDStr))
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		h.logger.Error("Invalid song ID", zap.String("song_id", songIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	err = h.svc.DeleteSong(songID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("Song not found", zap.Int("song_id", songID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
			return
		}
		h.logger.Error("Failed to delete song", zap.Error(err), zap.Int("song_id", songID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	h.logger.Info("Song deleted successfully", zap.Int("song_id", songID))
	c.JSON(http.StatusOK, gin.H{"message": "Song deleted"})
}

func (h *Handler) TruncateSongs(c *gin.Context) {
	h.logger.Info("Handling TruncateSongs request")
	h.logger.Debug("Initiating table truncation")
	err := h.svc.TruncateSongs()
	if err != nil {
		h.logger.Error("Failed to truncate songs table", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	h.logger.Info("Table truncated and sequence reset")
	c.JSON(http.StatusOK, gin.H{"message": "Table truncated and ID sequence reset"})
}
