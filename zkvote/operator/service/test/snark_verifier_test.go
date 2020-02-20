package test

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/operator/service/manager/voter"
)

var idCommitment = [...]string{
	"17610192990552485611214212559447309706539616482639833145108503521837267798810",
	"12911218845103750861406388355446622324232020446592892913783900846357287231484",
	"1032284940727177649939355598892149604970527830516654273792409518693495635491",
	"21176767283001926398440783773513762000716980892317227378236556477014800668400",
	"277593402026800763958336343436402082617452946079134938418951384937122081698",
	"6894328305305102806720810244282194911715263103973532827353421862495894269552",
	"16696521640980513762895570995527758356630582039153470423682046528515430261262",
	"3445457589949194301656936787185600418234571893114183450807142773072595456469",
	"312527539520761448694291090800047775034752705203482547162969903432178197468",
	"15339060964594858963739198162689168732368415427742964439329199487013648959013",
}

func TestRegister(t *testing.T) {
	id, _ := voter.NewIdentityPool()
	// New proposal
	p, _ := voter.NewProposal(id)
	qIdx := p.Propose("this is a question.")
	vkData, err := ioutil.ReadFile("../snark/verification_key.json")
	assert.Nil(t, err)

	y, n := 0, 0
	for i, s := range idCommitment {
		fmt.Println("")
		idc, _ := big.NewInt(0).SetString(s, 10)
		_, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
		assert.Nil(t, err)

		// submit proof
		dat, err := ioutil.ReadFile(fmt.Sprintf("./vectors/vote%d.proof", i))
		assert.Nil(t, err)

		err = p.VoteWithProof(qIdx, string(dat), string(vkData))
		assert.Nil(t, err)

		y, n = p.GetVotes(qIdx)
	}
	assert.Equal(t, 5, y)
	assert.Equal(t, 5, n)
}
