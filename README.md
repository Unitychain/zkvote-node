# kad_node: Using ZK-SNARK to Implement Decentralized Anonymous Voting on DHT
`kad_node` is a Kademlia DHT node. Nodes connect with each other to form a mesh network. On top of this network, nodes can propose and anyone could vote with zk. The votes could reveal anytime. (with zk, everyone knows the number of votes without revealing who votes)

## Flow
- Onboarding flow:
    - Run `kad_node`. The node will connect to other peers in the network.
        - Use defferent libp2p protocol
    - Nodes form two networks: DHT and pub/sub
    - The local running node provides a web UI for the user to register and submit proof.
    - __A pub/sub topic is a voting subject__
    - Node can __start a new subject (topic)__ or __discover a subject (topic)__
        - Use `Discovery` package to look up peers that are topic providers. See https://hackmd.io/@juincc/Hk_0PWFhS
        - Setup a stream from the joining node and to the topic providers
        - Topic provides respond their pub/sub topics
        - The joining node can either
            - Subscribe to one of the existing topic
            - Start a new topic, this will make the joining node become one of the topic provider
        - Attach a description to a topic
            - Store the description locally
            - key: topic, value: description
            - Respond to the requestor as well as the topic name
        - __Should be able to delegate the role of subject creator__
- Registering flow:
    - Subscribe a the topic `T`
    - Generate or import identity. This includes commitment and nullifier (Refer to semaphore)
    - Broadcast the hash of its identity to the peers from the same topic `T` so that the peers can index the identity.
    - Put the identity to the DHT
- Proving flow:
    - User generates the proof `P` that corresponds to the topic `T` via cli or web UI
    - Node broadcast the hash of the proof `P` so that the peers knows how many proofs are in `T` and index these proofs locally.
    - Node put the proof to the DHT
    - Proof message doesn't have to be signed
- Verifying Flow
    - Node collects the proofs from the DHT by looking up the indexed hash
    - Node verify the proof to see if it is valid
    - Node aggregates the result of the proofs

## Reference
- See [this document](https://hackmd.io/@juincc/B1QV5NN5S) for more details
