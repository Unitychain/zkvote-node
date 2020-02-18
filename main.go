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
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	"github.com/unitychain/zkvote-node/zkvote/node"
	zkvote "github.com/unitychain/zkvote-node/zkvote/operator"
)

func main() {
	path := flag.String("db", "node_data", "Database folder")
	serverPort := flag.Int("p", 9900, "Web UI port")
	cmds := flag.Bool("cmds", false, "Interactive commands")
	flag.Parse()

	utils.OpenLog()
	utils.LogInfo("======================")
	utils.LogInfo("===== Node Start =====")
	utils.LogInfo("======================")

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

	op, err := zkvote.NewOperator(ctx, ds, relay, bucketSize)
	if err != nil {
		panic(err)
	}

	server, err := restapi.NewServer(op, serverAddr)
	if err != nil {
		panic(err)
	}

	n := node.NewNode(ctx, ds)
	_ = n

	go server.ListenAndServe()
	fmt.Printf("HTTP server listens to port %d\n", *serverPort)

	if *cmds {
		op.Run()
	} else {
		select {}
	}
}
