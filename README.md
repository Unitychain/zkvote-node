# zkvote: Using ZK-SNARK to Implement Decentralized Anonymous Voting on p2p Network
**zkvote** is a powerful tool for anonymous voting. It uses a cryptographic function called ZK-SNARK to keep the voter from revealing its identity. It is also built on a peer-to-peer network so that so single entity or authortity can control the access or result of the voting. Moreover, zkvote utilizes a developing standard called Decentralized Identifier (DID) and Verifiable Credential (VC) to prove the validity of the identity.

## How it Works?
![](https://i.imgur.com/RAAnWn8.png)

1. **Generate keypair**
    - Use `zkvote-web` to generate keypair / DID / identity commitment
    - Doesn't support import at this moment
    - Store in browser localStorage
2. **Propose/Join**
    - Proposer
        - Use `zkvote-web` to propoese a new subject
    - Joiner
        - Use `zkvote-web` to join an existed subject
3. **Exchange VC**
    - Joiner sends its identity commitment to proposer via other network
      - For example: email, SMS...
    - Proposer receives the identity commitment of the joiner from the network
    - Proposer generates a verifiable credential (VC) for joiner and sends it to joiner. This VC includes:
        - Subject hash
        - identity commitment of joiner
        - Signature of issuer
        - public key of issuer
    - Joiner receives the VC
4. **Vote**
    - Proposer
        - Generates the proof that corresponds to the subject
        - Upload the proof and the VC to `zkvote-node`
    - Joiner
        - **Verify the VC for the subject**
        - Generates the proof that corresponds to the subject
        - Upload the proof and the VC to `zkvote-node`
5. **Open**
    - Use `zkvote-web` to see the latest result

## Contribution
See [this document](https://hackmd.io/@juincc/B1QV5NN5S) for more technical details
