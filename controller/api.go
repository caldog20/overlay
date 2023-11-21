package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
)

type APIServer struct {
	db     *DB
	router *gin.Engine
}

func Ping(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func NewAPIServer() *APIServer {
	db := NewDB()
	server := &APIServer{db: db}
	r := gin.Default()

	apiv1 := r.Group("/api/v1")

	{
		apiv1.POST("/login", server.Login)
		apiv1.GET("/ping", Ping)
	}

	server.router = r

	return server
}

func (server *APIServer) Run(ctx context.Context) error {
	go server.router.Run(":8080")
	<-ctx.Done()
	return nil
}

func (server *APIServer) Login(g *gin.Context) {
	login := struct {
		id     uint32 `json:"id"`
		pubkey string `json:"pubkey"`
	}{}
	if err := g.BindJSON(&login); err != nil {
		g.AbortWithStatus(http.StatusBadRequest)
		return
	}
	return login
}
