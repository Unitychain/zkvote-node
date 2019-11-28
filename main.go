package main

import (
	"context"
	"flag"
	"math/rand"
	"time"

	levelds "github.com/ipfs/go-ds-leveldb"
	zkvote "github.com/unitychain/zkvote-node/zkvote"
)

func main() {
	path := flag.String("db", "dht-data", "Database folder")
	flag.Parse()

	// ~~ 0c. Note that contexts are an ugly way of controlling component
	// lifecycles. Talk about the service-based host refactor.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set default values
	if *path == "" {
		*path = "node_data"
	}
	relay := false
	bucketSize := 1
	ds, err := levelds.NewDatastore(*path, nil)
	if err != nil {
		panic(err)
	}

	timeSeed := time.Now().UnixNano() / int64(time.Millisecond)
	rand.Seed(timeSeed)
	port := rand.Intn(100) + 10000

	node, err := zkvote.NewNode(ctx, ds, relay, bucketSize, port)
	if err != nil {
		panic(err)
	}
	node.Run()
}
