package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	size int
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, fmt.Errorf("failed to calculate size: %w", err)
}

func GinLogger(log *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		blw := &bodyLogWriter{ResponseWriter: c.Writer, size: 0}
		c.Writer = blw
		c.Next()
		duration := time.Since(start)

		log.Infow("Request:",
			"URI", c.Request.RequestURI,
			"Method", c.Request.Method,
			"StatusCode", strconv.Itoa(c.Writer.Status()),
			"Duration", duration.String(),
			"Size", strconv.Itoa(blw.size),
		)
	}
}
