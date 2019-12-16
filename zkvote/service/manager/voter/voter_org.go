package voter

import (
	"encoding/hex"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

type VoterOrg struct {
	*localContext.Context

	subjectHash   *subject.Hash
	Ps            *pubsub.PubSub
	Subscriptions map[subject.HashHex]*VoterSubscription
	messages      map[string][]*pubsub.Message
	identityIndex id.Index
}

type VoterSubscription struct {
	identitySubscription *pubsub.Subscription
	voteSubscription     *pubsub.Subscription
}

// GetIdentitySub ...
func (v *VoterSubscription) GetIdentitySub() *pubsub.Subscription {
	return v.identitySubscription
}

// GetVoteSub ...
func (v *VoterSubscription) GetVoteSub() *pubsub.Subscription {
	return v.voteSubscription
}

// NewVoterOrg ...
func NewVoterOrg(
	sHash *subject.Hash,
	ps *pubsub.PubSub,
	lc *localContext.Context,
) (*VoterOrg, error) {
	v := &VoterOrg{
		subjectHash:   sHash,
		Context:       lc,
		Ps:            ps,
		Subscriptions: make(map[subject.HashHex]*VoterSubscription),
		messages:      make(map[string][]*pubsub.Message),
		identityIndex: id.NewIndex(),
	}

	// Subscribe to two topics: one for identities, one for votes
	identitySub, err := v.Ps.Subscribe("identity/" + sHash.Hex().String())
	if err != nil {
		return nil, err
	}

	voteSub, err := v.Ps.Subscribe("vote/" + sHash.Hex().String())
	if err != nil {
		return nil, err
	}

	// Store the subscriptions
	VoterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	v.Subscriptions[sHash.Hex()] = VoterSub

	go identitySubHandler(v, sHash, identitySub)
	// go voteSubHandler(v, voteSub)

	return v, nil
}

// Register ...
func (v *VoterOrg) Register(identityCommitmentHex string) error {
	// Subscribe to pubsub
	identitySub, err := v.Ps.Subscribe("identity/" + v.subjectHash.Hex().String())
	if err != nil {
		return err
	}

	voteSub, err := v.Ps.Subscribe("vote/" + v.subjectHash.Hex().String())
	if err != nil {
		return err
	}

	// Store the subscriptions
	VoterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	v.Subscriptions[v.subjectHash.Hex()] = VoterSub

	// TODO: Why is this called 4 times?
	go identitySubHandler(v, v.subjectHash, identitySub)

	// TODO : Integrate identity_pool
	identity := id.NewIdentity(identityCommitmentHex)

	// Publish identity hash
	identityTopic := identitySub.Topic()
	v.Ps.Publish(identityTopic, identity.Hash().Byte())

	return nil
}

// Vote ...
func (v *VoterOrg) Vote() error {
	return nil
}

// Open ...
func (v *VoterOrg) Open() error {
	return nil
}

// GetSubscriptions .
// func (v *VoterOrg) GetSubscriptions() map[subject.HashHex]*VoterOrgSubscription {
// 	return v.subscriptions
// }

func pubsubHandler(v *VoterOrg, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		v.Mutex.Lock()
		msgs := v.messages[sub.Topic()]
		v.messages[sub.Topic()] = append(msgs, m)
		v.Mutex.Unlock()
	}
}

func identitySubHandler(v *VoterOrg, subjectHash *subject.Hash, subscription *pubsub.Subscription) {
	for {
		fmt.Println("identitySubHandler: Received message")

		m, err := subscription.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		identityHash := id.Hash(m.GetData())
		v.InsertIdentity(identityHash)

		// v.Mutex.Lock()
		// identityHash := identity.Hash(m.GetData())

		// fmt.Println("identitySubHandler: Received message")

		// identityHashSet := v.Cache.GetAIDIndex(string(subjectHash.Hex()))
		// if nil == identityHashSet {
		// 	identityHashSet = identity.NewHashSet()
		// }
		// identityHashSet[identityHash.Hex()] = "ID"
		// v.Cache.InsertIDIndex(string(subjectHash.Hex()), identityHashSet)
		// v.Mutex.Unlock()
	}
}

func voteSubHandler(v *VoterOrg, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		v.Mutex.Lock()
		msgs := v.messages[sub.Topic()]
		v.messages[sub.Topic()] = append(msgs, m)
		v.Mutex.Unlock()
	}
}

// InsertIDIndex .
func (v *VoterOrg) InsertIDIndex(k subject.HashHex, set id.HashSet) {
	v.identityIndex[k] = set
}

//GetIDIndexes .
func (v *VoterOrg) GetIDIndexes() id.Index {
	return v.identityIndex
}

// GetIdentityHashSet ...
func (v *VoterOrg) GetIdentityHashSet() id.HashSet {
	return v.identityIndex[v.subjectHash.Hex()]
}

// InsertIdentity .
func (v *VoterOrg) InsertIdentity(identityHash id.Hash) {
	// m.Mutex.Lock()
	identityHashSet := v.GetIdentityHashSet()
	if nil == identityHashSet {
		identityHashSet = id.NewHashSet()
	}
	identityHashSet[identityHash.Hex()] = "ID"
	v.InsertIDIndex(v.subjectHash.Hex(), identityHashSet)
	// m.Mutex.Unlock()
}

// GetIdentityHashes ...
func (v *VoterOrg) GetIdentityHashes() []id.Hash {
	identityHashSet := v.GetIdentityHashSet()
	if nil == identityHashSet {
		identityHashSet = id.NewHashSet()
	}
	list := make([]id.Hash, 0)
	for hx := range identityHashSet {
		h, err := hex.DecodeString(hx.String())
		if err != nil {
			fmt.Println(err)
		}
		list = append(list, id.Hash(h))
	}
	return list
}

// Broadcast ...
func (v *VoterOrg) Broadcast() error {
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

	return v.Ps.Publish(topic, []byte(data))
}

// PrintInboundMessages ...
func (v *VoterOrg) PrintInboundMessages() error {
	v.Mutex.RLock()
	topics := make([]string, 0, len(v.messages))
	for k := range v.messages {
		topics = append(topics, k)
	}
	v.Mutex.RUnlock()

	s := promptui.Select{
		Label: "topic",
		Items: topics,
	}

	_, topic, err := s.Run()
	if err != nil {
		return err
	}

	v.Mutex.Lock()
	defer v.Mutex.Unlock()
	for _, m := range v.messages[topic] {
		fmt.Printf("<<< from: %s >>>: %s\n", m.GetFrom(), string(m.GetData()))
	}
	v.messages[topic] = nil
	return nil
}
