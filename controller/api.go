package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (c *Controller) register(g *gin.Context) {
	remote := g.RemoteIP()
	remote = strings.Split(remote, ":")[0]
	remote = fmt.Sprintf("%s:%d", remote, 5555)
	node, _ := c.addNode(g.Param("hostname"), remote)
	g.IndentedJSON(http.StatusCreated, node)
}

func (c *Controller) deregister(g *gin.Context) {
	id := g.Param("id")
	err := c.deleteNode(id)
	if err != nil {
		g.AbortWithStatus(http.StatusNotFound)
	} else {
		g.Status(http.StatusOK)
	}
}

//func listNodes(c *gin.Context) {
//	nodes := getNodeList()
//	c.IndentedJSON(http.StatusOK, nodes)
//}

func (c *Controller) getNode(g *gin.Context) {
	ip := g.Param("vpnip")
	n, err := c.getNodeByIP(ip)
	if err != nil {
		g.AbortWithStatus(http.StatusNotFound)
		return
	}
	g.IndentedJSON(http.StatusOK, n)
}

func (c *Controller) Serve(ctx context.Context) {
	router := gin.Default()

	apiGroup := router.Group("/api")

	{
		//apiGroup.GET("/nodes", listNodes)
		apiGroup.GET("/node/:vpnip", c.getNode)
		apiGroup.GET("register/:hostname", c.register)
		apiGroup.GET("/deregister/:id", c.deregister)
	}
	router.NoRoute()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			return
		}
	}()

	select {
	case <-ctx.Done():
		srv.Shutdown(ctx)
		return
	}

}
