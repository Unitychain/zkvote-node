package operator

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	ipns "github.com/ipfs/go-ipns"

	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"
	msdnDiscovery "github.com/libp2p/go-libp2p/p2p/discovery"

	"github.com/manifoldco/promptui"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/unitychain/zkvote-node/zkvote/common/store"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	"github.com/unitychain/zkvote-node/zkvote/operator/service/manager"
	"github.com/unitychain/zkvote-node/zkvote/operator/service/utils"
)

// node client version

const DB_PEER_ID = "peerID"

// Node ...
type Operator struct {
	*localContext.Context
	*manager.Manager
	dht       *dht.IpfsDHT
	pubsub    *pubsub.PubSub
	db        datastore.Batching
	mdnsPeers map[peer.ID]peer.AddrInfo
	streams   chan network.Stream
}

func loadPrivateKey(ds datastore.Batching) (crypto.PrivKey, error) {
	var prvKey crypto.PrivKey
	var err error

	tmpStore, _ := store.NewStore(nil, ds)
	strKey, _ := tmpStore.GetLocal(DB_PEER_ID)
	if 0 == len(strKey) {
		utils.LogInfof("Generate a new private key!")
		prvKey, _, _ = crypto.GenerateKeyPair(crypto.ECDSA, 0)
		// priv, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		b, _ := prvKey.Raw()
		tmpStore.PutLocal(DB_PEER_ID, utils.Remove0x(utils.GetHexStringFromBytes(b)))
	} else {
		prvKey, err = crypto.UnmarshalECDSAPrivateKey(utils.GetBytesFromHexString(strKey))
		if err != nil {
			return nil, fmt.Errorf("unmarshal private key error, %v", err)
		}
	}
	b, _ := prvKey.GetPublic().Bytes()
	utils.LogInfof("peer pub key: %v", utils.GetHexStringFromBytes(b))

	return prvKey, err
}

// NewNode create a new node with its implemented protocols
func NewOperator(ctx context.Context, ds datastore.Batching, relay bool, bucketSize int) (*Operator, error) {
	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

	// Ignoring most errors for brevity
	// See echo example for more details and better implementation

	prvKey, err := loadPrivateKey(ds)
	if err != nil {
		panic(err)
	}
	// listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))

	opts := []libp2p.Option{libp2p.ConnectionManager(cmgr), libp2p.Identity(prvKey)}
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

	d1, err := dht.New(context.Background(), host, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(store.NodeValidator{}))
	if err != nil {
		panic(err)
	}

	// Pubsub
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	op := &Operator{
		dht:       d1,
		pubsub:    ps,
		db:        ds,
		mdnsPeers: make(map[peer.ID]peer.AddrInfo),
		streams:   make(chan network.Stream, 128),
	}
	s, _ := store.NewStore(d1, ds)
	cache, _ := store.NewCache()
	op.Context = localContext.NewContext(new(sync.RWMutex), host, s, cache, &ctx)

	vkData, err := ioutil.ReadFile("./snark/verification_key.json")
	if err != nil {
		panic(err)
	}
	op.Manager, _ = manager.NewManager(ps, d1, op.Context, string(vkData))

	mdns, err := msdnDiscovery.NewMdnsService(ctx, host, time.Second*5, "")
	if err != nil {
		panic(err)
	}
	mdns.RegisterNotifee(op)

	return op, nil
}

// HandlePeerFound msdn handler
func (o *Operator) HandlePeerFound(pi peer.AddrInfo) {
	o.Mutex.Lock()
	o.mdnsPeers[pi.ID] = pi
	o.Mutex.Unlock()

	if err := o.Context.Host.Connect(*o.Ctx, pi); err != nil {
		fmt.Printf("failed to connect to mDNS peer: %s\n", err)
	}
}

// Info ...
func (o *Operator) Info() error {
	// 0b. Let's get a sense of what those defaults are. What transports are we
	// listening on? Each transport will have a multiaddr. If you run this
	// multiple times, you will get different port numbers. Note how we listen
	// on all interfaces by default.
	// fmt.Println("My addresses:")
	// for _, a := range o.h.Addrs() {
	// 	fmt.Printf("\t%s\n", a)
	// }

	fmt.Println()
	fmt.Println("My peer ID:")
	fmt.Printf("\t%s\n", o.Host.ID())

	fmt.Println()
	fmt.Println("My identified multiaddrs:")
	for _, a := range o.Host.Addrs() {
		fmt.Printf("\t%s/p2p/%s\n", a, o.Host.ID())
	}

	// What protocols are added by default?
	// fmt.Println()
	// fmt.Println("Protocols:")
	// for _, p := range o.h.Mux().Protocols() {
	// 	fmt.Printf("\t%s\n", p)
	// }

	// What peers do we have in our peerstore? (hint: we've connected to nobody so far).
	fmt.Println()
	fmt.Println("Peers in peerstore:")
	for _, p := range o.Host.Peerstore().PeersWithAddrs() {
		fmt.Printf("\t%s\n", p)
	}
	fmt.Println(len(o.Host.Peerstore().PeersWithAddrs()))

	// DHT routing table
	fmt.Println("DHT Routing table:")
	o.dht.RoutingTable().Print()

	// Connections
	fmt.Println("Connections:")
	fmt.Println(len(o.Host.Network().Conns()))

	fmt.Println("Created subjectHashHex:")
	for sh := range o.Cache.GetCreatedSubjects() {
		fmt.Println(sh)
	}

	fmt.Println("Collected subjects:")
	fmt.Println(o.Cache.GetCollectedSubjects())

	fmt.Println("Subcribed topics:")
	fmt.Println(o.pubsub.GetTopics())

	fmt.Println("Identity Index:")
	fmt.Println(o.GetIdentityIndex())

	fmt.Println("Voter Identity Merkle Tree Contents:")
	fmt.Println(o.GetVoterIdentities())

	fmt.Println("Ballot Map:")
	fmt.Println(o.GetBallotMaps())

	return nil
}

// DHTBootstrap ...
func (o *Operator) DHTBootstrap(seeds ...ma.Multiaddr) error {
	fmt.Println("Will bootstrap for 30 seconds...")

	ctx, cancel := context.WithTimeout(*o.Ctx, 30*time.Second)
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
			if err := o.Host.Connect(ctx, ai); err != nil {
				fmt.Printf("Failed while connecting to peer: %s; %s\n", ai, err)
			} else {
				fmt.Printf("Succeeded while connecting to peer: %s\n", ai)
			}
		}(*ai)
	}

	wg.Wait()

	// if err := o.dht.BootstrapRandom(ctx); err != nil && err != context.DeadlineExceeded {
	// 	return fmt.Errorf("failed while bootstrapping DHT: %w", err)
	// }
	if err := o.dht.Bootstrap(ctx); err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("failed while bootstrapping DHT: %w", err)
	}

	fmt.Println("bootstrap OK! Routing table:")
	o.dht.RoutingTable().Print()

	return nil
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

// Run interactive commands
func (o *Operator) Run() {
	commands := []struct {
		name string
		exec func() error
	}{
		{"My info", o.handleMyInfo},
		{"Manager: Propose a subject", o.handlePropose},
		{"Manager: Join a subject", o.handleJoin},
		{"Manager: Find topic providers", o.handleFindProposers},
		{"Manager: Collect all topics", o.handleCollect},
		// {"Manager: Sync identity index", o.handleSyncIdentityIndex},
		{"Store: Put DHT", o.handlePutDHT},
		{"Store: Get DHT", o.handleGetDHT},
		{"Store: Put Local", o.handlePutLocal},
		{"Store: Get Local", o.handleGetLocal},
		{"DHT: Bootstrap (all seeds)", o.handleDHTBootstrap},
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

func (o *Operator) handleMyInfo() error {
	return o.Info()
}

func (o *Operator) handleDHTBootstrap() error {
	return o.DHTBootstrap(dht.DefaultBootstrapPeers...)
}

func (o *Operator) handlePutDHT() error {
	// return o.Store.PutDHT()
	return nil
}

func (o *Operator) handlePutLocal() error {
	// return o.Store.PutLocal()
	return nil
}

func (o *Operator) handleGetDHT() error {
	// return o.Store.GetDHT()
	return nil
}

func (o *Operator) handleGetLocal() error {
	// return o.Store.GetLocal()
	return nil
}

func (o *Operator) handlePropose() error {
	p := promptui.Prompt{
		Label: "Subject title",
	}
	title, err := p.Run()
	if err != nil {
		return err
	}

	p = promptui.Prompt{
		Label: "Subject description",
	}
	description, err := p.Run()
	if err != nil {
		return err
	}

	return o.Propose(title, description, "")
}

func (o *Operator) handleJoin() error {
	p := promptui.Prompt{
		Label: "Subject hash hex",
	}
	subjectHashHex, err := p.Run()
	if err != nil {
		return err
	}

	p = promptui.Prompt{
		Label: "Identity commitment hex",
	}
	identityCommitmentHex, err := p.Run()
	if err != nil {
		return err
	}

	return o.Join(subjectHashHex, identityCommitmentHex)
}

// func (o *Operator) handleSyncIdentityIndex() error {
// 	return o.SyncIdentityIndex()
// }

// func (o *Operator) handlePrintInboundMessages() error {
// 	return o.PrintInboundMessages()
// }

func (o *Operator) handleFindProposers() error {
	peers, err := o.FindProposers()
	if err != nil {
		fmt.Println(err)
	}

	for p := range peers {
		fmt.Println("found peer", p)
		o.SetProvider(p.ID, "")
	}

	return nil
}

func (o *Operator) handleCollect() error {
	// subjects, err := o.Collect()
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// for subject := range subjects {
	// 	fmt.Println("Collect: ", subject)
	// }

	return nil
}
