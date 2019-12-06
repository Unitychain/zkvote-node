package zkvote

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
	"github.com/ipfs/go-datastore"
	ipns "github.com/ipfs/go-ipns"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"
	msdnDiscovery "github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/manifoldco/promptui"
	ma "github.com/multiformats/go-multiaddr"
	subject "github.com/unitychain/zkvote-node/zkvote/pb"
)

// node client version
const clientVersion = "zkvote/0.0.1"

// Node ...
type Node struct {
	sync.RWMutex
	host.Host
	*Collector
	*Voter
	*Store
	ctx       context.Context
	dht       *dht.IpfsDHT
	pubsub    *pubsub.PubSub
	db        datastore.Batching
	mdnsPeers map[peer.ID]peer.AddrInfo
	streams   chan network.Stream
}

// NewNode create a new node with its implemented protocols
func NewNode(ctx context.Context, ds datastore.Batching, relay bool, bucketSize int, port int) (*Node, error) {
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

	d2, err := dht.New(context.Background(), host, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: host.Peerstore()},
	}))
	_ = d2

	// Use an empty validator here for simplicity
	// CAUTION! Use d2 will cause a "stream reset" error!

	d1, err := dht.New(context.Background(), host, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(NodeValidator{}))
	if err != nil {
		panic(err)
	}

	// Pubsub
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	node := &Node{
		ctx:       ctx,
		Host:      host,
		dht:       d1,
		pubsub:    ps,
		db:        ds,
		mdnsPeers: make(map[peer.ID]peer.AddrInfo),
		streams:   make(chan network.Stream, 128),
	}

	node.Collector, err = NewCollector(node)
	node.Voter, err = NewVoter(node)
	node.Store, err = NewStore(node)

	mdns, err := msdnDiscovery.NewMdnsService(ctx, host, time.Second*5, "")
	if err != nil {
		panic(err)
	}
	mdns.RegisterNotifee(node)

	return node, nil
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

	fmt.Println("Created subjectHashHex:")
	for sh := range node.createdSubjects {
		fmt.Println(sh)
	}

	fmt.Println("Collected subjects:")
	fmt.Println(node.collectedSubjects)

	fmt.Println("IdentityIndex:")
	for k, set := range node.identityIndex.Index {
		fmt.Println(k)
		fmt.Println(set)
	}

	fmt.Println("Subcribed topics:")
	fmt.Println(node.pubsub.GetTopics())

	fmt.Println("Subscriptions:")
	for key, value := range node.subscriptions {
		fmt.Println(key)
		fmt.Println(value.identitySubscription)
	}

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

// Run ...
func (node *Node) Run() {
	commands := []struct {
		name string
		exec func() error
	}{
		{"My info", node.handleMyInfo},
		{"DHT: Bootstrap (all seeds)", node.handleDHTBootstrap},
		{"Store: Put DHT", node.handlePutDHT},
		{"Store: Get DHT", node.handleGetDHT},
		{"Store: Put Local", node.handlePutLocal},
		{"Store: Get Local", node.handleGetLocal},
		{"Voter: Propose a subject", node.handlePropose},
		{"Voter: Join a subject", node.handleJoin},
		{"Voter: Register identity", node.handleRegister},
		{"Voter: Publish a message", node.handleBroadcast},
		{"Voter: Sync identity index", node.handleSyncIdentityIndex},
		{"Voter: Print inbound messages", node.handlePrintInboundMessages},
		{"Collector: Advertise topic", node.handleAnnounce},
		{"Collector: Find topic providers", node.handleFindProposers},
		{"Collector: Collect all topics", node.handleCollect},
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

func (node *Node) handleDHTBootstrap() error {
	return node.DHTBootstrap(dht.DefaultBootstrapPeers...)
}

func (node *Node) handlePutDHT() error {
	return node.PutDHT()
}

func (node *Node) handlePutLocal() error {
	return node.PutLocal()
}

func (node *Node) handleGetDHT() error {
	return node.GetDHT()
}

func (node *Node) handleGetLocal() error {
	return node.GetLocal()
}

func (node *Node) handleJoin() error {
	p := promptui.Prompt{
		Label: "Subject hash hex",
	}
	subjectHashHex, err := p.Run()
	if err != nil {
		return err
	}

	return node.Join(subjectHashHex)
}

func (node *Node) handlePropose() error {
	p := promptui.Prompt{
		Label: "Subject title",
	}
	subjectTitle, err := p.Run()
	if err != nil {
		return err
	}

	return node.Propose(subjectTitle)
}

func (node *Node) handleRegister() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Subject hash hex: ")
	scanner.Scan()
	subjectHashHex := scanner.Text()
	fmt.Println("Subject hash hex: ", subjectHashHex)

	fmt.Print("Identity commitment hex: ")
	scanner.Scan()
	identityCommitmentHex := scanner.Text()
	fmt.Println("Input value: ", identityCommitmentHex)

	return node.Register(subjectHashHex, identityCommitmentHex)
}

func (node *Node) handleBroadcast() error {
	return node.Broadcast()
}

func (node *Node) handleSyncIdentityIndex() error {
	return node.SyncIdentityIndex()
}

func (node *Node) handlePrintInboundMessages() error {
	return node.PrintInboundMessages()
}

func (node *Node) handleAnnounce() error {
	return node.Announce()
}

func (node *Node) handleFindProposers() error {
	return node.FindProposers()
}

func (node *Node) handleCollect() error {
	return node.Collect()
}
