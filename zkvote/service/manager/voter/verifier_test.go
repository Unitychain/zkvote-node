package voter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/unitychain/zkvote-node/zkvote/model/ballot"
)

const proof = `{"root":"8689527539353720499147190441125863458163151721613317120863488267899702105029","nullifier_hash":"2816172240193667514175132752992643557697925698066061870859177551815944593050","proof":{"pi_a":["8691763105990886350963363271989552223474436289023125494979621270764800459232","7959638426364130174789515688875030651662144363587569706716497148362191159938","1"],"pi_b":[["19672397907949105136004182724814129941981129347700701231981910882913270766720","7453112622360304714622374664770379760730765585344569137151987786509397694750"],["17713299595964412202149265033076504022773960690349222314724526374801920909426","9367986521030447807907551596620994546862391690182193686900834575186052682863"],["1","0"]],"pi_c":["10492872918931104924574489517332635617291002221774517162909731671120506042533","18385559078235175716438291260368371366388578425177984394233419600921259384571","1"],"protocol":"groth"},"public_signal":["8689527539353720499147190441125863458163151721613317120863488267899702105029","2816172240193667514175132752992643557697925698066061870859177551815944593050","43379584054787486383572605962602545002668015983485933488536749112829893476306","9695771177025341492834515246141576816221841749730679787621778614635855226700"]}`
const vk = `{
	"protocol": "groth",
	"nPublic": 4,
	"IC": [
	 [
	  "12418545593032500588698912709858281690104918105591274283537109113822330018376",
	  "20931904325052726909555873356162419291111683474434199586234246405351358007076",
	  "1"
	 ],
	 [
	  "20628397838886466824304774917282912326347021119195946673626155488764554827426",
	  "18932651013819139346274151857743429939893289761710026454765280889796912696842",
	  "1"
	 ],
	 [
	  "9430082519347770932300688456187927018669734780076138387824045158783443793264",
	  "19035983126452889389177117681190772422040879526183631526295743625992416117803",
	  "1"
	 ],
	 [
	  "5339279297631183081611477158443448611463339794273824317466285743222921054272",
	  "21057965667085293282174454954502607167023462692201738053860693743225240761449",
	  "1"
	 ],
	 [
	  "11475864085691081945964608346166222023070665352547086256716493162333077451565",
	  "15302677909953538721119124466918030428018783463721966486770201550923965579503",
	  "1"
	 ]
	],
	"vk_alfa_1": [
	 "11417303501339734522061829883131690884464687546132933239999994469274434090237",
	 "20801625657706405044121202841268227255984709948858400596558408013277081990093",
	 "1"
	],
	"vk_beta_2": [
	 [
	  "15257490107441725059621677412864036480078865260063490946726373442889551238440",
	  "9020580066257313919895951867317841042871677370976861797204795731633729171698"
	 ],
	 [
	  "21334844345637983632423046114150286621525704308071587861080421838729635646372",
	  "2765545994978726708626986051583881165994278094222716471116706295294794922202"
	 ],
	 [
	  "1",
	  "0"
	 ]
	],
	"vk_gamma_2": [
	 [
	  "19299205038058634043829352835539955376769981433970440115917884366814859006292",
	  "8342416061208732362064350898041247602669567628279882768725386578305247719325"
	 ],
	 [
	  "3442260395583412909421013007569773427454826955388410891893726285013570287982",
	  "8361267789454297508215707315326381019461303200294243196889158571277886539726"
	 ],
	 [
	  "1",
	  "0"
	 ]
	],
	"vk_delta_2": [
	 [
	  "5062182384391342192337991328239217874430714296685330421031162504423846795081",
	  "15850203067089548369752564423001943762450669412562262968241788942611647064783"
	 ],
	 [
	  "20839684184298635835821096117900598379047635241034635840926363422579779728129",
	  "11438468695066608516884166617488045468172155993365350530015168114983584945886"
	 ],
	 [
	  "1",
	  "0"
	 ]
	],
	"vk_alfabeta_12": [
	 [
	  [
	   "1771370623011861096283290486633417905479293005029315111081408028652703459029",
	   "12914249198801283864300209970317578031883024384357293584908854148710526229065"
	  ],
	  [
	   "8958020601366913821397254479696202425755572472670047355017483735522103210457",
	   "20751269435344791505547470314310668628545745173873280868656671941079743778014"
	  ],
	  [
	   "15702126836192116074530195648947531023153582603146758933420631213344387899819",
	   "1662909296365194722479908744193152065915598802598776435900971669257072537718"
	  ]
	 ],
	 [
	  [
	   "188008691152688957740767049704383346935187291417385932698251564691849341039",
	   "12816162447555779195804823867928366175328084184030453349996517432007728863345"
	  ],
	  [
	   "11237632839848656569396308683432285994371584434311292682559261560863943019074",
	   "16382436169025775978479796332218873275691062872334788512550110225069525255474"
	  ],
	  [
	   "3622773912027958778557570061634500555588971011803724052062044732977297513461",
	   "2716869227587311165419358441136455979986628100666908995776034320784110241968"
	  ]
	 ]
	]
   }
`

func TestParse(t *testing.T) {
	b, _ := NewBallot(proof)
	assert.Equal(t, "8689527539353720499147190441125863458163151721613317120863488267899702105029", b.Root)
	assert.Equal(t, "2816172240193667514175132752992643557697925698066061870859177551815944593050", b.NullifierHash)
	assert.Equal(t, 4, len(b.PublicSignal))
	assert.Equal(t, "8689527539353720499147190441125863458163151721613317120863488267899702105029", b.PublicSignal[0])
	assert.Equal(t, "8691763105990886350963363271989552223474436289023125494979621270764800459232", b.Proof.PiA[0])
}

func TestVerify(t *testing.T) {
	b, _ := NewBallot(proof)
	assert.True(t, Verify(vk, b.Proof, b.PublicSignal))
}
