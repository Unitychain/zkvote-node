package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	levelds "github.com/ipfs/go-ds-leveldb"
	"github.com/unitychain/zkvote-node/restapi"
	zkvote "github.com/unitychain/zkvote-node/zkvote/service"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

func main() {
	path := flag.String("db", "node_data", "Database folder")
	serverPort := flag.Int("p", 9900, "Web UI port")
	cmds := flag.Bool("cmds", false, "Interactive commands")
	flag.Parse()

	utils.OpenLog()

	// ~~ 0c. Note that contexts are an ugly way of controlling component
	// lifecycles. Talk about the service-based host refactor.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set default values
	if *path == "" {
		*path = "node_data"
	}
	*path = "data/" + *path

	relay := false
	bucketSize := 1
	ds, err := levelds.NewDatastore(*path, nil)
	if err != nil {
		panic(err)
	}

	timeSeed := time.Now().UnixNano() / int64(time.Millisecond)
	rand.Seed(timeSeed)
	// p2pPort := rand.Intn(100) + 10000
	// serverPort := strconv.Itoa(rand.Intn(100) + 3000)
	serverAddr := ":" + strconv.Itoa(*serverPort)

	node, err := zkvote.NewNode(ctx, ds, relay, bucketSize)
	if err != nil {
		panic(err)
	}

	server, err := restapi.NewServer(node, serverAddr)
	if err != nil {
		panic(err)
	}

	go server.ListenAndServe()
	fmt.Printf("HTTP server listens to port %d\n", *serverPort)

	if *cmds {
		node.Run()
	} else {
		select {}
	}
}
