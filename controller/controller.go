package controller

import "github.com/gin-gonic/gin"

type IController interface {
	Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc)
	Close() error
}
