package head

import (
	"fmt"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// Service ...
type Service struct {
	conf      Config
	directory string

	log *slog.Logger
}

// NewService ...
func NewService(log *slog.Logger, conf Config, directory string) *Service {
	return &Service{
		conf:      conf,
		directory: directory,
		log:       log,
	}
}

// Start ...
func (s *Service) Start() error {
	if !s.conf.Enabled {
		return nil
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.GET("/heads/:xuid", s.fetchHead)

	if err := router.Run(s.conf.Address); err != nil {
		return err
	}
	return nil
}

// fetchHead ...
func (s *Service) fetchHead(ctx *gin.Context) {
	xuid := ctx.Param("xuid")
	if xuid == "" || strings.ContainsAny(xuid, "/\\..") {
		ctx.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"error": fmt.Sprintf("invalid xuid provided: %s", xuid)})
		return
	}

	file, err := os.Open(filepath.Join(s.directory, xuid+".png"))
	if os.IsNotExist(err) {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "head not found"})
		return
	}
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to open head file: %s", err.Error()),
		})
		return
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to decode head image: %s", err.Error()),
		})
		return
	}

	ctx.Header("Content-Type", "image/png")
	ctx.Header("Cache-Control", "public, max-age=60")

	if err = png.Encode(ctx.Writer, img); err != nil {
		s.log.Error("failed to encode head to writer", "error", err, "xuid", xuid)
	}
}
