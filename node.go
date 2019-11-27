package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"log"

	ggio "github.com/gogo/protobuf/io"
	proto "github.com/gogo/protobuf/proto"
	ds "github.com/ipfs/go-datastore"
	ipns "github.com/ipfs/go-ipns"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"
	msdnDiscovery "github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/manifoldco/promptui"
	ma "github.com/multiformats/go-multiaddr"
	subject "github.com/unitychain/kad_node/pb"
)

// node client version
const clientVersion = "kad_node/0.0.1"

// Node ...
type Node struct {
	sync.RWMutex

	host.Host
	ctx       context.Context
	dht       *dht.IpfsDHT
	discovery discovery.Discovery
	pubsub    *pubsub.PubSub

	mdnsPeers       map[peer.ID]peer.AddrInfo
	messages        map[string][]*pubsub.Message
	streams         chan network.Stream
	providers       map[peer.ID]string
	allTopics       map[string]string
	createdSubjects map[string]string

	*SubjectProtocol // subject protocol impl
}

// NewNode create a new node with its implemented protocols
func NewNode(ctx context.Context, ds ds.Batching, relay bool, bucketSize int, port int) (*Node, error) {
	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

	// Ignoring most errors for brevity
	// See echo example for more details and better implementation

	// priv, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	priv, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))

	opts := []libp2p.Option{libp2p.ConnectionManager(cmgr), libp2p.Identity(priv), libp2p.ListenAddrs(listen)}
	if relay {
		opts = append(opts, libp2p.EnableRelay(circuit.OptHop))
	}

	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		panic(err)
	}

	d1, err := dht.New(context.Background(), host, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: host.Peerstore()},
	}))
	_ = d1

	// Use an empty validator here for simplicity
	d2, err := dht.New(context.Background(), host, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(KNValidator{}))
	if err != nil {
		panic(err)
	}

	// Discovery
	rd := routingDiscovery.NewRoutingDiscovery(d2)

	// Pubsub
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	node := &Node{
		ctx:       ctx,
		Host:      host,
		dht:       d2,
		discovery: rd,
		mdnsPeers: make(map[peer.ID]peer.AddrInfo),
		messages:  make(map[string][]*pubsub.Message),
		streams:   make(chan network.Stream, 128),
		pubsub:    ps,
		providers: make(map[peer.ID]string),
		allTopics: make(map[string]string),
	}

	done := make(chan bool, 1)
	node.SubjectProtocol = NewSubjectProtocol(node, done)

	mdns, err := msdnDiscovery.NewMdnsService(ctx, host, time.Second*5, "")
	if err != nil {
		panic(err)
	}
	mdns.RegisterNotifee(node)

	return node, nil
}

func (node *Node) handler(s network.Stream) {
	fmt.Printf("*** Got a new chat stream from %s! ***\n", s.Conn().RemotePeer())
	node.streams <- s
}

// HandlePeerFound msdn handler
func (node *Node) HandlePeerFound(pi peer.AddrInfo) {
	node.Lock()
	node.mdnsPeers[pi.ID] = pi
	node.Unlock()

	if err := node.Connect(node.ctx, pi); err != nil {
		fmt.Printf("failed to connect to mDNS peer: %s\n", err)
	}
}

// Info ...
func (node *Node) Info() error {
	// 0b. Let's get a sense of what those defaults are. What transports are we
	// listening on? Each transport will have a multiaddr. If you run this
	// multiple times, you will get different port numbers. Note how we listen
	// on all interfaces by default.
	// fmt.Println("My addresses:")
	// for _, a := range node.h.Addrs() {
	// 	fmt.Printf("\t%s\n", a)
	// }

	fmt.Println()
	fmt.Println("My peer ID:")
	fmt.Printf("\t%s\n", node.ID())

	fmt.Println()
	fmt.Println("My identified multiaddrs:")
	for _, a := range node.Addrs() {
		fmt.Printf("\t%s/p2p/%s\n", a, node.ID())
	}

	// What protocols are added by default?
	// fmt.Println()
	// fmt.Println("Protocols:")
	// for _, p := range node.h.Mux().Protocols() {
	// 	fmt.Printf("\t%s\n", p)
	// }

	// What peers do we have in our peerstore? (hint: we've connected to nobody so far).
	fmt.Println()
	fmt.Println("Peers in peerstore:")
	for _, p := range node.Peerstore().PeersWithAddrs() {
		fmt.Printf("\t%s\n", p)
	}
	fmt.Println(len(node.Peerstore().PeersWithAddrs()))

	// DHT routing table
	fmt.Println("DHT Routing table:")
	node.dht.RoutingTable().Print()

	// Connections
	fmt.Println("Connections:")
	fmt.Println(len(node.Network().Conns()))

	// All created sucjects
	fmt.Println("All created subjects:")
	fmt.Println(node.allTopics)

	return nil
}

// DHTBootstrap ...
func (node *Node) DHTBootstrap(seeds ...ma.Multiaddr) error {
	fmt.Println("Will bootstrap for 30 seconds...")

	ctx, cancel := context.WithTimeout(node.ctx, 30*time.Second)
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
			if err := node.Connect(ctx, ai); err != nil {
				fmt.Printf("Failed while connecting to peer: %s; %s\n", ai, err)
			} else {
				fmt.Printf("Succeeded while connecting to peer: %s\n", ai)
			}
		}(*ai)
	}

	wg.Wait()

	// if err := node.dht.BootstrapRandom(ctx); err != nil && err != context.DeadlineExceeded {
	// 	return fmt.Errorf("failed while bootstrapping DHT: %w", err)
	// }
	if err := node.dht.Bootstrap(ctx); err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("failed while bootstrapping DHT: %w", err)
	}

	fmt.Println("bootstrap OK! Routing table:")
	node.dht.RoutingTable().Print()

	return nil
}

// NearestPeersToQuery ...
func (node *Node) NearestPeersToQuery() error {
	ctx := context.Background()

	fmt.Print("Key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	ps, err := node.dht.GetClosestPeers(ctx, k)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Number of peers: ", len(ps)+1)

	p := <-ps
	fmt.Println("First peer: ", p)

	return nil
}

// PutValue ...
func (node *Node) PutValue() error {
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

	err := node.dht.PutValue(ctx, k, vb)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// GetValue ...
func (node *Node) GetValue() error {
	ctx := context.Background()

	fmt.Print("Key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	vb, err := node.dht.GetValue(ctx, k)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Value: ", string(vb))

	return nil
}

// SubscribeToTopic ...
func (node *Node) SubscribeToTopic() error {
	p := promptui.Prompt{
		Label: "topic name",
	}
	topic, err := p.Run()
	if err != nil {
		return err
	}

	sub, err := node.pubsub.Subscribe(topic)
	if err != nil {
		return err
	}

	go pubsubHandler(node, sub)

	return nil
}

// PublishToTopic ...
func (node *Node) PublishToTopic() error {
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

	return node.pubsub.Publish(topic, []byte(data))
}

// PrintInboundMessages ...
func (node *Node) PrintInboundMessages() error {
	node.RLock()
	topics := make([]string, 0, len(node.messages))
	for k := range node.messages {
		topics = append(topics, k)
	}
	node.RUnlock()

	s := promptui.Select{
		Label: "topic",
		Items: topics,
	}

	_, topic, err := s.Run()
	if err != nil {
		return err
	}

	node.Lock()
	defer node.Unlock()
	for _, m := range node.messages[topic] {
		fmt.Printf("<<< from: %s >>>: %s\n", m.GetFrom(), string(m.GetData()))
	}
	node.messages[topic] = nil
	return nil
}

func pubsubHandler(node *Node, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(node.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		node.Lock()
		msgs := node.messages[sub.Topic()]
		node.messages[sub.Topic()] = append(msgs, m)
		node.Unlock()
	}
}

// TopicAdvertise ...
func (node *Node) TopicAdvertise() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Before advertising, make sure the host has a subscription
	if len(node.pubsub.GetTopics()) != 0 {
		_, err := node.discovery.Advertise(ctx, "subjects", routingDiscovery.TTL(10*time.Minute))
		return err
	}
	return fmt.Errorf("kad node hasn't subscribed to any topic")
}

// FindTopicProviders ...
func (node *Node) FindTopicProviders() error {
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

// CollectAllTopics ...
func (node *Node) CollectAllTopics() error {
	for p := range node.providers {
		// Ignore self ID
		if p == node.ID() {
			continue
		}
		node.GetCreatedSubjects(p)
	}

	return nil
}

// ListTopics ...
func (node *Node) ListTopics() error {
	topics := node.pubsub.GetTopics()
	fmt.Println(topics)

	return nil
}

// NewMessageData helper method - generate message data shared between all node's p2p protocols
// messageId: unique for requests, copied from request for responses
func (node *Node) NewMessageData(messageID string, gossip bool) *subject.MessageData {
	// Add protobufs bin data for message author public key
	// this is useful for authenticating  messages forwarded by a node authored by another node
	nodePubKey, err := node.Peerstore().PubKey(node.ID()).Bytes()

	if err != nil {
		panic("Failed to get public key for sender from local peer store.")
	}

	return &subject.MessageData{ClientVersion: clientVersion,
		NodeId:     peer.IDB58Encode(node.ID()),
		NodePubKey: nodePubKey,
		Timestamp:  time.Now().Unix(),
		Id:         messageID,
		Gossip:     gossip}
}

// helper method - writes a protobuf go data object to a network stream
// data: reference of protobuf go data object to send (not the object itself)
// s: network stream to write the data to
func (node *Node) sendProtoMessage(id peer.ID, p protocol.ID, data proto.Message) bool {
	s, err := node.NewStream(context.Background(), id, p)
	if err != nil {
		log.Println(err)
		return false
	}
	writer := ggio.NewFullWriter(s)
	err = writer.WriteMsg(data)
	if err != nil {
		log.Println(err)
		s.Reset()
		return false
	}
	// FullClose closes the stream and waits for the other side to close their half.
	err = helpers.FullClose(s)
	if err != nil {
		log.Println(err)
		s.Reset()
		return false
	}
	return true
}

// // Authenticate incoming p2p message
// // message: a protobufs go data object
// // data: common p2p message data
// func (n *Node) authenticateMessage(message proto.Message, data *p2p.MessageData) bool {
// 	// store a temp ref to signature and remove it from message data
// 	// sign is a string to allow easy reset to zero-value (empty string)
// 	sign := data.Sign
// 	data.Sign = nil

// 	// marshall data without the signature to protobufs3 binary format
// 	bin, err := proto.Marshal(message)
// 	if err != nil {
// 		log.Println(err, "failed to marshal pb message")
// 		return false
// 	}

// 	// restore sig in message data (for possible future use)
// 	data.Sign = sign

// 	// restore peer id binary format from base58 encoded node id data
// 	peerId, err := peer.IDB58Decode(data.NodeId)
// 	if err != nil {
// 		log.Println(err, "Failed to decode node id from base58")
// 		return false
// 	}

// 	// verify the data was authored by the signing peer identified by the public key
// 	// and signature included in the message
// 	return n.verifyData(bin, []byte(sign), peerId, data.NodePubKey)
// }

// // sign an outgoing p2p message payload
// func (n *Node) signProtoMessage(message proto.Message) ([]byte, error) {
// 	data, err := proto.Marshal(message)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return n.signData(data)
// }

// // sign binary data using the local node's private key
// func (n *Node) signData(data []byte) ([]byte, error) {
// 	key := n.Peerstore().PrivKey(n.ID())
// 	res, err := key.Sign(data)
// 	return res, err
// }

// Verify incoming p2p message data integrity
// data: data to verify
// signature: author signature provided in the message payload
// peerId: author peer id from the message payload
// pubKeyData: author public key from the message payload
// func (n *Node) verifyData(data []byte, signature []byte, peerId peer.ID, pubKeyData []byte) bool {
// 	key, err := crypto.UnmarshalPublicKey(pubKeyData)
// 	if err != nil {
// 		log.Println(err, "Failed to extract key from message key data")
// 		return false
// 	}

// 	// extract node id from the provided public key
// 	idFromKey, err := peer.IDFromPublicKey(key)

// 	if err != nil {
// 		log.Println(err, "Failed to extract peer id from public key")
// 		return false
// 	}

// 	// verify that message author node id matches the provided node public key
// 	if idFromKey != peerId {
// 		log.Println(err, "Node id and provided public key mismatch")
// 		return false
// 	}

// 	res, err := key.Verify(data, signature)
// 	if err != nil {
// 		log.Println(err, "Error authenticating data")
// 		return false
// 	}

// 	return res
// }
