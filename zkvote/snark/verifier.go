package snark

import (
	"encoding/json"

	goSnarkVerifier "github.com/arnaucube/go-snark/externalVerif"
	"github.com/arnaucube/go-snark/groth16"
	goSnarkUtils "github.com/arnaucube/go-snark/utils"

	"github.com/unitychain/zkvote-node/zkvote/common/utils"
)

// VerifyByFile : verify proof
// func VerifyByFile(vkPath string, pfPath string) bool {

// 	dat, err := ioutil.ReadFile(pfPath)
// 	ballot, err := ba.NewBallot(string(dat))
// 	if err != nil {
// 		return false
// 	}

// 	vkFile, err := ioutil.ReadFile(vkPath)
// 	if err != nil {
// 		return false
// 	}
// 	return Verify(string(vkFile), ballot.Proof, ballot.PublicSignal)
// }

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
		utils.LogErrorf("GrothVkFromString error: %s", err.Error())
		return false
	}
	// utils.LogInfof("vk parsed: %v", vk)

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
		utils.LogErrorf("GrothProofFromString error: %s\n", err.Error())
		return false
	}
	// fmt.Println("proof parsed:", grothProof)

	//
	// public signals
	//
	publicSignals, err := goSnarkUtils.ArrayStringToBigInt(publicSignal)
	if err != nil {
		utils.LogErrorf("ArrayStringToBigInt error: %s\n", err.Error())
		return false
	}
	utils.LogDebugf("publicSignals parsed: %v", publicSignals)

	verified := groth16.VerifyProof(vk, grothProof, publicSignals, true)
	return verified
}
