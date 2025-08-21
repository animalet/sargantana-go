package controller

import (
	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

type IController interface {
	Bind(engine *gin.Engine, config config.Config, loginMiddleware gin.HandlerFunc)
	Close() error
}
