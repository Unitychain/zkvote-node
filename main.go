package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

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
	type_operator := flag.Bool("op", true, "activate as an operator")
	type_node := flag.Bool("n", false, "activate as a node")
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

	if *type_node {
		n := node.NewNode(ctx, ds, bucketSize)
		_ = n
	} else if *type_operator {
		serverAddr := ":" + strconv.Itoa(*serverPort)

		op, err := zkvote.NewOperator(ctx, ds, relay, bucketSize)
		if err != nil {
			panic(err)
		}

		server, err := restapi.NewServer(op, serverAddr)
		if err != nil {
			panic(err)
		}

		go server.ListenAndServe()
		fmt.Printf("HTTP server listens to port %d\n", *serverPort)

		if *cmds {
			op.Run()
		} else {
			select {}
		}
	}
}
