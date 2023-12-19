package node

import (
	"context"
	"github.com/caldog20/go-overlay/node"
	"log"
)

func main() {
	node, err := node.NewNode("5000", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}

	node.Run(context.TODO())
}
