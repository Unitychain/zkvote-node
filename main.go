package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	levelds "github.com/ipfs/go-ds-leveldb"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/manifoldco/promptui"
	"github.com/multiformats/go-multiaddr"
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

	node, err := NewNode(ctx, ds, relay, bucketSize, port)
	if err != nil {
		panic(err)
	}
	node.Run()
}

// Run ...
func (node *Node) Run() {
	commands := []struct {
		name string
		exec func() error
	}{
		{"My info", node.handleMyInfo},
		{"DHT: Bootstrap (all seeds)", func() error { return node.handleDHTBootstrap(dht.DefaultBootstrapPeers...) }},
		{"DHT: Find nearest peers to query", node.handleNearestPeersToQuery},
		{"DHT: Put value", node.handlePutValue},
		{"DHT: Get value", node.handleGetValue},
		{"Discovery: Advertise topic", node.handleTopicAdvertise},
		{"Discovery: Find topic providers", node.handleFindTopicProviders},
		{"Pubsub: Subscribe to topic", node.handleSubscribeToTopic},
		{"Pubsub: Publish a message", node.handlePublishToTopic},
		{"Pubsub: Print inbound messages", node.handlePrintInboundMessages},
		{"Pubsub: Collect all topics", node.handleCollectAllTopics},
		{"Pubsub: List subscribed topics", node.handleListTopics},
	}

	var str []string
	for _, c := range commands {
		str = append(str, c.name)
	}

	for {
		sel := promptui.Select{
			Label: "What do you want to do?",
			Items: str,
			Size:  1000,
		}

		fmt.Println()
		i, _, err := sel.Run()
		if err != nil {
			panic(err)
		}

		if err := commands[i].exec(); err != nil {
			fmt.Printf("command failed: %s\n", err)
		}
	}
}

func (node *Node) handleMyInfo() error {
	return node.Info()
}

func (node *Node) handleDHTBootstrap(seeds ...multiaddr.Multiaddr) error {
	return node.DHTBootstrap(seeds...)
}

func (node *Node) handleNearestPeersToQuery() error {
	return node.NearestPeersToQuery()
}

func (node *Node) handlePutValue() error {
	return node.PutValue()
}

func (node *Node) handleGetValue() error {
	return node.GetValue()
}

func (node *Node) handleSubscribeToTopic() error {
	return node.SubscribeToTopic()
}

func (node *Node) handlePublishToTopic() error {
	return node.PublishToTopic()
}

func (node *Node) handlePrintInboundMessages() error {
	return node.PrintInboundMessages()
}

func (node *Node) handleTopicAdvertise() error {
	return node.TopicAdvertise()
}

func (node *Node) handleFindTopicProviders() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	peers, err := node.discovery.FindPeers(ctx, "subjects")
	if err != nil {
		return err
	}

	for p := range peers {
		fmt.Println("found peer", p)
		node.Peerstore().AddAddrs(p.ID, p.Addrs, 24*time.Hour)
		node.providers[p.ID] = ""
	}

	fmt.Println("Subject creators: ")
	fmt.Println(node.providers)
	return err
}

func (node *Node) handleCollectAllTopics() error {
	return node.CollectAllTopics()
}

func (node *Node) handleListTopics() error {
	return node.ListTopics()
}
