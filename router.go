package gobase

import (
	"github.com/gin-gonic/gin"
	"strings"
)

type Router struct {
	Method string
	Path string
	Handler gin.HandlerFunc
}

type Routers []Router

func InitRouter(e *gin.Engine, routers Routers)  {
	for _, r := range routers {
		if strings.EqualFold("any", r.Method) {
			e.Any(r.Path, r.Handler)
			continue
		}
		e.Handle(r.Method, r.Path, r.Handler)
	}
}