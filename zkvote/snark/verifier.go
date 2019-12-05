package snark

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/arnaucube/go-snark/externalVerif"
	goSnarkVerifier "github.com/arnaucube/go-snark/externalVerif"
	"github.com/arnaucube/go-snark/groth16"
	goSnarkUtils "github.com/arnaucube/go-snark/utils"
)

type Vote struct {
	Root          string                     `json:"root"`
	NullifierHash string                     `json:"nullifier_hash"`
	Proof         *externalVerif.CircomProof `json:"proof"`
	PublicSignal  []string                   `json:"public_signal"` //root, nullifiers_hash, signal_hash, external_nullifier
}

// Parse : parse proof from json string to Vote struct
func Parse(jsonProof string) *Vote {

	var vote Vote
	err := json.Unmarshal([]byte(jsonProof), &vote)
	if err != nil {
		fmt.Println("unmarshal", err)
	}
	// fmt.Println(vote)
	return &vote
}

// VerifyByFile : verify proof
func VerifyByFile(vkPath string, pfPath string) bool {

	dat, err := ioutil.ReadFile("./vote.proof")
	proof := Parse(string(dat))

	vkFile, err := ioutil.ReadFile(vkPath)
	if err != nil {
		return false
	}
	return Verify(string(vkFile), proof.Proof, proof.PublicSignal)
}

// Verify : verify proof
func Verify(vkString string, proof *goSnarkVerifier.CircomProof, publicSignal []string) bool {

	//
	// verification key
	//
	var circomVk goSnarkVerifier.CircomVk
	err := json.Unmarshal([]byte(vkString), &circomVk)
	if err != nil {
		return false
	}

	var strVk goSnarkUtils.GrothVkString
	strVk.IC = circomVk.IC
	strVk.G1.Alpha = circomVk.Alpha1
	strVk.G2.Beta = circomVk.Beta2
	strVk.G2.Gamma = circomVk.Gamma2
	strVk.G2.Delta = circomVk.Delta2
	vk, err := goSnarkUtils.GrothVkFromString(strVk)
	if err != nil {
		fmt.Printf("GrothVkFromString error: %s\n", err.Error())
		return false
	}
	// fmt.Println("vk parsed:", vk)

	//
	// proof
	//
	strProof := goSnarkUtils.GrothProofString{
		PiA: proof.PiA,
		PiB: proof.PiB,
		PiC: proof.PiC,
	}
	grothProof, err := goSnarkUtils.GrothProofFromString(strProof)
	if err != nil {
		fmt.Printf("GrothProofFromString error: %s\n", err.Error())
		return false
	}
	// fmt.Println("proof parsed:", grothProof)

	//
	// public signals
	//
	publicSignals, err := goSnarkUtils.ArrayStringToBigInt(publicSignal)
	if err != nil {
		fmt.Printf("ArrayStringToBigInt error: %s\n", err.Error())
		return false
	}
	fmt.Println("publicSignals parsed:", publicSignals)

	verified := groth16.VerifyProof(vk, grothProof, publicSignals, true)
	return verified
}
