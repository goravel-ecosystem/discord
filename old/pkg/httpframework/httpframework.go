package httpframework

import (
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var (
	router *gin.Engine
	once   sync.Once
)

func Init(middlewares ...gin.HandlerFunc) {
	once.Do(func() {
		env := os.Getenv("APP_ENV")
		if env == "prod" || env == "production" {
			gin.SetMode(gin.ReleaseMode)
		}

		router = gin.New()
	})
}

// GetInstance instance of router
func GetInstance() *gin.Engine {
	if router == nil {
		log.Fatal().Msg("Router not initialized")
	}

	return router
}
