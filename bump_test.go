package bc

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	jsonExample = `{"blockHeight":814435,"path":[[{"offset":20,"hash":"0dc75b4efeeddb95d8ee98ded75d781fcf95d35f9d88f7f1ce54a77a0c7c50fe"},{"offset":21,"txid":true,"hash":"3ecead27a44d013ad1aae40038acbb1883ac9242406808bb4667c15b4f164eac"}],[{"offset":11,"hash":"5745cf28cd3a31703f611fb80b5a080da55acefa4c6977b21917d1ef95f34fbc"}],[{"offset":4,"hash":"522a096a1a6d3b64a4289ab456134158d8443f2c3b8ed8618bd2b842912d4b57"}],[{"offset":3,"hash":"191c70d2ecb477f90716d602f4e39f2f81f686f8f4230c255d1b534dc85fa051"}],[{"offset":0,"hash":"1f487b8cd3b11472c56617227e7e8509b44054f2a796f33c52c28fd5291578fd"}],[{"offset":1,"hash":"5ecc0ad4f24b5d8c7e6ec5669dc1d45fcb3405d8ce13c0860f66a35ef442f562"}],[{"offset":1,"hash":"31631241c8124bc5a9531c160bfddb6fcff3729f4e652b10d57cfd3618e921b1"}]]}`

	hexExample           = `fe636d0c0007021400fe507c0c7aa754cef1f7889d5fd395cf1f785dd7de98eed895dbedfe4e5bc70d1502ac4e164f5bc16746bb0868404292ac8318bbac3800e4aad13a014da427adce3e010b00bc4ff395efd11719b277694cface5aa50d085a0bb81f613f70313acd28cf4557010400574b2d9142b8d28b61d88e3b2c3f44d858411356b49a28a4643b6d1a6a092a5201030051a05fc84d531b5d250c23f4f886f6812f9fe3f402d61607f977b4ecd2701c19010000fd781529d58fc2523cf396a7f25440b409857e7e221766c57214b1d38c7b481f01010062f542f45ea3660f86c013ced80534cb5fd4c19d66c56e7e8c5d4bf2d40acc5e010100b121e91836fd7cd5102b654e9f72f3cf6fdbfd0b161c53a9c54b12c841126331`
	rootExample          = `bb6f640cc4ee56bf38eb5a1969ac0c16caa2d3d202b22bf3735d10eec0ca6e00`
	txidExample          = `3ecead27a44d013ad1aae40038acbb1883ac9242406808bb4667c15b4f164eac`
	rootOfBlockTxExample = `1a1e779cd7dfc59f603b4e88842121001af822b2dc5d3b167ae66152e586a6b0`
	fakeMadeUpNum        = 814435
)

var blockTxExample = []string{
	"b6d4d13aa08bb4b6cdb3b329cef29b5a5d55d85a85c330d56fddbce78d99c7d6",
	"426f65f6a6ce79c909e54d8959c874a767db3076e76031be70942b896cc64052",
	"adc23d36cc457d5847968c2e4d5f017a6f12a2f165102d10d2843f5276cfe68e",
	"728714bbbddd81a54cae473835ae99eb92ed78191327eb11a9d7494273dcad2a",
	"e3aa0230aa81abd483023886ad12790acf070e2a9f92d7f0ae3bebd90a904361",
	"4848b9e94dd0e4f3173ebd6982ae7eb6b793de305d8450624b1d86c02a5c61d9",
	"912f77eefdd311e24f96850ed8e701381fc4943327f9cf73f9c4dec0d93a056d",
	"397fe2ae4d1d24efcc868a02daae42d1b419289d9a1ded3a5fe771efcc1219d9",
}

func TestNewBUMPFromMerkleTree(t *testing.T) {
	merkles, err := BuildMerkleTreeStore(blockTxExample)
	require.NoError(t, err)
	fmt.Println(merkles)
	bump, err := NewBUMPFromMerkleTree(fakeMadeUpNum, merkles)
	require.NoError(t, err)
	bytes, err := json.MarshalIndent(bump, "", "  ")
	require.NoError(t, err)
	fmt.Println(string(bytes))
	for _, txid := range blockTxExample {
		root, err := bump.CalculateRootGivenTxid(txid)
		require.NoError(t, err)
		require.Equal(t, rootOfBlockTxExample, root)
	}
}

func TestNewBUMPFromStr(t *testing.T) {
	bump, err := NewBUMPFromStr(hexExample)
	require.NoError(t, err)
	str, err := bump.String()
	require.NoError(t, err)
	require.Equal(t, hexExample, str)
}

func TestNewBUMPFromJson(t *testing.T) {
	jBump, err := NewBUMPFromJSON(jsonExample)
	require.NoError(t, err)
	jStr, err := jBump.String()
	require.NoError(t, err)
	require.Equal(t, hexExample, jStr)
}

func TestCalculateRootGivenTxid(t *testing.T) {
	bump, err := NewBUMPFromJSON(jsonExample)
	require.NoError(t, err)
	root, err := bump.CalculateRootGivenTxid(txidExample)
	require.NoError(t, err)
	require.Equal(t, rootExample, root)
}
