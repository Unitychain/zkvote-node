package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"

	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
	msdnDiscovery "github.com/libp2p/go-libp2p/p2p/discovery"

	ds "github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	ipns "github.com/ipfs/go-ipns"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/manifoldco/promptui"
	"github.com/multiformats/go-multiaddr"
)

// KadNode ...
type KadNode struct {
	sync.RWMutex

	ctx       context.Context
	h         host.Host
	dht       *dht.IpfsDHT
	discovery discovery.Discovery
	pubsub    *pubsub.PubSub

	mdnsPeers map[peer.ID]peer.AddrInfo
	messages  map[string][]*pubsub.Message
	streams   chan network.Stream
	providers map[peer.ID]string
	allTopics map[string]string
}

func main() {
	path := flag.String("db", "dht-data", "Database folder")
	flag.Parse()

	// ~~ 0c. Note that contexts are an ugly way of controlling component
	// lifecycles. Talk about the service-based host refactor.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// kn, err := makeAndStartNode(ctx)

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

	kn, err := makeAndStartNode(ctx, ds, relay, bucketSize)
	// kn, err := _makeAndStartNode(ctx)
	if err != nil {
		panic(err)
	}

	kn.Run()
}

func _makeAndStartNode(ctx context.Context) (*KadNode, error) {
	var kaddht *dht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		kaddht, err = dht.New(ctx, h)
		return kaddht, err
	}

	// 0a. Let's build a new libp2p host. The New constructor uses functional
	// parameters. You don't need to provide any parameters. libp2p comes with
	// sane defaults OOTB, but in order to stay slim, we don't attach a routing
	// implementation by default. Let's do that.
	host, err := libp2p.New(ctx, libp2p.Routing(newDHT))
	if err != nil {
		panic(err)
	}

	mdns, err := msdnDiscovery.NewMdnsService(ctx, host, time.Second*5, "")
	if err != nil {
		panic(err)
	}

	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	kn := &KadNode{
		ctx:       ctx,
		h:         host,
		dht:       kaddht,
		mdnsPeers: make(map[peer.ID]peer.AddrInfo),
		messages:  make(map[string][]*pubsub.Message),
		streams:   make(chan network.Stream, 128),
		pubsub:    ps,
	}

	host.SetStreamHandler(protocol.ID("/taipei/chat/2019"), kn.handler)

	mdns.RegisterNotifee(kn)

	return kn, nil
}

var bootstrapDone int64

func makeAndStartNode(ctx context.Context, ds ds.Batching, relay bool, bucketSize int) (*KadNode, error) {
	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

	priv, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, 0)

	opts := []libp2p.Option{libp2p.ConnectionManager(cmgr), libp2p.Identity(priv)}
	if relay {
		opts = append(opts, libp2p.EnableRelay(circuit.OptHop))
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		panic(err)
	}

	d1, err := dht.New(context.Background(), h, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: h.Peerstore()},
	}))
	_ = d1

	// Use an empty validator here for simplicity
	d2, err := dht.New(context.Background(), h, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(KNValidator{}))
	if err != nil {
		panic(err)
	}

	// Discovery
	rd := routingDiscovery.NewRoutingDiscovery(d2)

	// Pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	kn := &KadNode{
		ctx:       ctx,
		h:         h,
		dht:       d2,
		discovery: rd,
		mdnsPeers: make(map[peer.ID]peer.AddrInfo),
		messages:  make(map[string][]*pubsub.Message),
		streams:   make(chan network.Stream, 128),
		pubsub:    ps,
		providers: make(map[peer.ID]string),
		allTopics: make(map[string]string),
	}

	h.SetStreamHandler(protocol.ID("/taipei/chat/2019"), kn.handler)

	// Topic Query Handler
	h.SetStreamHandler("/topicquery/1.0.0", kn.handleTopicQuery)

	mdns, err := msdnDiscovery.NewMdnsService(ctx, h, time.Second*5, "")
	if err != nil {
		panic(err)
	}
	mdns.RegisterNotifee(kn)

	return kn, nil
}

func (kn *KadNode) handler(s network.Stream) {
	fmt.Printf("*** Got a new chat stream from %s! ***\n", s.Conn().RemotePeer())
	kn.streams <- s
}

// HandlePeerFound ...
func (kn *KadNode) HandlePeerFound(pi peer.AddrInfo) {
	kn.Lock()
	kn.mdnsPeers[pi.ID] = pi
	kn.Unlock()

	if err := kn.h.Connect(kn.ctx, pi); err != nil {
		fmt.Printf("failed to connect to mDNS peer: %s\n", err)
	}
}

// handleTopicQuery
func (kn *KadNode) handleTopicQuery(s network.Stream) {
	fmt.Printf("*** Got a topic query from %s! ***\n", s.Conn().RemotePeer())
	// List subscribed topics
	topics := kn.pubsub.GetTopics()
	// Make message
	m := strings.Join(topics, ",")

	fmt.Println(m)
	// Writer
	writer := bufio.NewWriter(s)
	writer.WriteString(m + "\n")
	writer.Flush()
}

// Run ...
func (kn *KadNode) Run() {
	commands := []struct {
		name string
		exec func() error
	}{
		{"My info", kn.handleMyInfo},
		{"DHT: Bootstrap (all seeds)", func() error { return kn.handleDHTBootstrap(dht.DefaultBootstrapPeers...) }},
		{"DHT: Bootstrap (3 random seeds)", func() error { return kn._handleDHTBootstrap() }},
		{"DHT: Find nearest peers to query", kn.handleNearestPeersToQuery},
		{"DHT: Put value", kn.handlePutValue},
		{"DHT: Get value", kn.handleGetValue},
		{"Discovery: Advertise topic", kn.handleTopicAdvertise},
		{"Discovery: Find topic providers", kn.handleFindTopicProviders},
		{"Pubsub: Subscribe to topic", kn.handleSubscribeToTopic},
		{"Pubsub: Publish a message", kn.handlePublishToTopic},
		{"Pubsub: Print inbound messages", kn.handlePrintInboundMessages},
		{"Pubsub: Collect all topics", kn.handleCollectAllTopics},
		{"Pubsub: List subscribed topics", kn.handleListTopics},
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

func (kn *KadNode) handleMyInfo() error {
	// 0b. Let's get a sense of what those defaults are. What transports are we
	// listening on? Each transport will have a multiaddr. If you run this
	// multiple times, you will get different port numbers. Note how we listen
	// on all interfaces by default.
	// fmt.Println("My addresses:")
	// for _, a := range kn.h.Addrs() {
	// 	fmt.Printf("\t%s\n", a)
	// }

	fmt.Println()
	fmt.Println("My peer ID:")
	fmt.Printf("\t%s\n", kn.h.ID())

	// fmt.Println()
	// fmt.Println("My identified multiaddrs:")
	// for _, a := range kn.h.Addrs() {
	// 	fmt.Printf("\t%s/p2p/%s\n", a, kn.h.ID())
	// }

	// What protocols are added by default?
	// fmt.Println()
	// fmt.Println("Protocols:")
	// for _, p := range kn.h.Mux().Protocols() {
	// 	fmt.Printf("\t%s\n", p)
	// }

	// What peers do we have in our peerstore? (hint: we've connected to nobody so far).
	fmt.Println()
	fmt.Println("Peers in peerstore:")
	for _, p := range kn.h.Peerstore().PeersWithAddrs() {
		fmt.Printf("\t%s\n", p)
		addr := kn.h.Peerstore().PeerInfo(p)
		fmt.Printf("\t%s\n", addr)
	}
	fmt.Println(len(kn.h.Peerstore().PeersWithAddrs()))

	// DHT routing table
	fmt.Println("DHT Routing table:")
	kn.dht.RoutingTable().Print()

	// Connections
	fmt.Println("Connections:")
	fmt.Println(len(kn.h.Network().Conns()))

	return nil
}

func (kn *KadNode) handleDHTBootstrap(seeds ...multiaddr.Multiaddr) error {
	fmt.Println("Will bootstrap for 30 seconds...")

	ctx, cancel := context.WithTimeout(kn.ctx, 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(seeds))

	for _, ma := range seeds {
		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return err
		}

		go func(ai peer.AddrInfo) {
			defer wg.Done()

			fmt.Printf("Connecting to peer: %s\n", ai)
			if err := kn.h.Connect(ctx, ai); err != nil {
				fmt.Printf("Failed while connecting to peer: %s; %s\n", ai, err)
			} else {
				fmt.Printf("Succeeded while connecting to peer: %s\n", ai)
			}
		}(*ai)
	}

	wg.Wait()

	// if err := kn.dht.BootstrapRandom(ctx); err != nil && err != context.DeadlineExceeded {
	// 	return fmt.Errorf("failed while bootstrapping DHT: %w", err)
	// }
	if err := kn.dht.Bootstrap(ctx); err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("failed while bootstrapping DHT: %w", err)
	}

	fmt.Println("bootstrap OK! Routing table:")
	kn.dht.RoutingTable().Print()

	return nil
}

func (kn *KadNode) _handleDHTBootstrap() error {
	// bootstrap in the background
	// it's safe to start doing this _before_ establishing any connections
	// as we'll trigger a boostrap round as soon as we get a connection (?)
	// anyways.

	bootstrapConcurency := 3
	limiter := make(chan struct{}, bootstrapConcurency)

	go func() {
		if limiter != nil {
			limiter <- struct{}{}
		}

		// for i := 0; i < 10; i++ {
		// 	if err := kn.h.Connect(context.Background(), bootstrapper()); err != nil {
		// 		fmt.Println("===bootstrap connect failed: ", err)
		// 		i--
		// 	}
		// }

		if err := kn.h.Connect(context.Background(), bootstrapper()); err != nil {
			fmt.Println("bootstrap connect failed: ", err)
		}

		if limiter != nil {
			<-limiter
		}
		atomic.AddInt64(&bootstrapDone, 1)

	}()

	// kn.dht.Bootstrap(context.Background())
	// kn.dht.RoutingTable().Print()

	return nil
}

func bootstrapper() pstore.PeerInfo {
	addr := dht.DefaultBootstrapPeers[rand.Intn(len(dht.DefaultBootstrapPeers))]
	ai, err := pstore.InfoFromP2pAddr(addr)
	if err != nil {
		panic(err)
	}

	return *ai
}

func (kn *KadNode) handleNearestPeersToQuery() error {
	ctx := context.Background()

	fmt.Print("Key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	ps, err := kn.dht.GetClosestPeers(ctx, k)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Number of peers: ", len(ps)+1)

	p := <-ps
	fmt.Println("First peer: ", p)

	return nil
}

func (kn *KadNode) handlePutValue() error {
	ctx := context.Background()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Key: ")
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	fmt.Print("Value: ")
	scanner.Scan()
	v := scanner.Text()
	vb := []byte(v)
	fmt.Println("Input value: ", v)

	err := kn.dht.PutValue(ctx, k, vb)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func (kn *KadNode) handleGetValue() error {
	ctx := context.Background()

	fmt.Print("Key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	vb, err := kn.dht.GetValue(ctx, k)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Value: ", string(vb))

	return nil
}

func (kn *KadNode) handleSubscribeToTopic() error {
	p := promptui.Prompt{
		Label: "topic name",
	}
	topic, err := p.Run()
	if err != nil {
		return err
	}

	sub, err := kn.pubsub.Subscribe(topic)
	if err != nil {
		return err
	}

	go pubsubHandler(kn, sub)

	return nil
}

func (kn *KadNode) handlePublishToTopic() error {
	p := promptui.Prompt{
		Label: "topic name",
	}
	topic, err := p.Run()
	if err != nil {
		return err
	}

	p = promptui.Prompt{
		Label: "data",
	}
	data, err := p.Run()
	if err != nil {
		return err
	}

	return kn.pubsub.Publish(topic, []byte(data))
}

func (kn *KadNode) handlePrintInboundMessages() error {
	kn.RLock()
	topics := make([]string, 0, len(kn.messages))
	for k := range kn.messages {
		topics = append(topics, k)
	}
	kn.RUnlock()

	s := promptui.Select{
		Label: "topic",
		Items: topics,
	}

	_, topic, err := s.Run()
	if err != nil {
		return err
	}

	kn.Lock()
	defer kn.Unlock()
	for _, m := range kn.messages[topic] {
		fmt.Printf("<<< from: %s >>>: %s\n", m.GetFrom(), string(m.GetData()))
	}
	kn.messages[topic] = nil
	return nil
}

func pubsubHandler(kn *KadNode, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(kn.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		kn.Lock()
		msgs := kn.messages[sub.Topic()]
		kn.messages[sub.Topic()] = append(msgs, m)
		kn.Unlock()
	}
}

func (kn *KadNode) handleTopicAdvertise() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Before advertising, make sure the host has a subscription
	if len(kn.pubsub.GetTopics()) != 0 {
		_, err := kn.discovery.Advertise(ctx, "topic", routingDiscovery.TTL(10*time.Minute))
		return err
	}
	return fmt.Errorf("kad node hasn't subscribed to any topic")
}

func (kn *KadNode) handleFindTopicProviders() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	peers, err := kn.discovery.FindPeers(ctx, "topic")
	if err != nil {
		return err
	}

	for p := range peers {
		fmt.Println("found peer", p)
		kn.h.Peerstore().AddAddrs(p.ID, p.Addrs, 24*time.Hour)
		kn.providers[p.ID] = ""
	}

	fmt.Println("Topic Providers: ")
	fmt.Println(kn.providers)
	return err
}

func (kn *KadNode) handleCollectAllTopics() error {
	// Dial to every topic prividers
	for p := range kn.providers {
		// Ignore self ID
		if p == kn.h.ID() {
			continue
		}
		s, err := kn.h.NewStream(context.Background(), p, "/topicquery/1.0.0")
		if err != nil {
			fmt.Println(err)
		}

		// Reader
		reader := bufio.NewReader(s)
		str, _ := reader.ReadString('\n')
		// Remove \n
		str = strings.TrimSuffix(str, "\n")

		// Store all topics
		for _, s := range strings.Split(str, ",") {
			kn.allTopics[s] = ""
		}
		fmt.Println(kn.allTopics)
	}

	return nil
}

// handleTopicQuery
func (kn *KadNode) handleListTopics() error {
	topics := kn.pubsub.GetTopics()
	fmt.Println(topics)

	return nil
}
