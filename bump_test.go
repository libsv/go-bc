package bc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	jsonExample  = `{"blockHeight":814435,"path":[{"20":{"hash":"0dc75b4efeeddb95d8ee98ded75d781fcf95d35f9d88f7f1ce54a77a0c7c50fe"},"21":{"hash":"3ecead27a44d013ad1aae40038acbb1883ac9242406808bb4667c15b4f164eac","txid":true}},{"11":{"hash":"5745cf28cd3a31703f611fb80b5a080da55acefa4c6977b21917d1ef95f34fbc"}},{"4":{"hash":"522a096a1a6d3b64a4289ab456134158d8443f2c3b8ed8618bd2b842912d4b57"}},{"3":{"hash":"191c70d2ecb477f90716d602f4e39f2f81f686f8f4230c255d1b534dc85fa051"}},{"0":{"hash":"1f487b8cd3b11472c56617227e7e8509b44054f2a796f33c52c28fd5291578fd"}},{"1":{"hash":"5ecc0ad4f24b5d8c7e6ec5669dc1d45fcb3405d8ce13c0860f66a35ef442f562"}},{"1":{"hash":"31631241c8124bc5a9531c160bfddb6fcff3729f4e652b10d57cfd3618e921b1"}}]}`
	jsonExample2 = `{"blockHeight":814435,"path":[[{"offset":20,"hash":"0dc75b4efeeddb95d8ee98ded75d781fcf95d35f9d88f7f1ce54a77a0c7c50fe"},{"offset":21,"txid":true,"hash":"3ecead27a44d013ad1aae40038acbb1883ac9242406808bb4667c15b4f164eac"}],[{"offset":11,"hash":"5745cf28cd3a31703f611fb80b5a080da55acefa4c6977b21917d1ef95f34fbc"}],[{"offset":4,"hash":"522a096a1a6d3b64a4289ab456134158d8443f2c3b8ed8618bd2b842912d4b57"}],[{"offset":3,"hash":"191c70d2ecb477f90716d602f4e39f2f81f686f8f4230c255d1b534dc85fa051"}],[{"offset":0,"hash":"1f487b8cd3b11472c56617227e7e8509b44054f2a796f33c52c28fd5291578fd"}],[{"offset":1,"hash":"5ecc0ad4f24b5d8c7e6ec5669dc1d45fcb3405d8ce13c0860f66a35ef442f562"}],[{"offset":1,"hash":"31631241c8124bc5a9531c160bfddb6fcff3729f4e652b10d57cfd3618e921b1"}]]}`
	hexExample   = `fe636d0c0007021400fe507c0c7aa754cef1f7889d5fd395cf1f785dd7de98eed895dbedfe4e5bc70d1502ac4e164f5bc16746bb0868404292ac8318bbac3800e4aad13a014da427adce3e010b00bc4ff395efd11719b277694cface5aa50d085a0bb81f613f70313acd28cf4557010400574b2d9142b8d28b61d88e3b2c3f44d858411356b49a28a4643b6d1a6a092a5201030051a05fc84d531b5d250c23f4f886f6812f9fe3f402d61607f977b4ecd2701c19010000fd781529d58fc2523cf396a7f25440b409857e7e221766c57214b1d38c7b481f01010062f542f45ea3660f86c013ced80534cb5fd4c19d66c56e7e8c5d4bf2d40acc5e010100b121e91836fd7cd5102b654e9f72f3cf6fdbfd0b161c53a9c54b12c841126331`
	hexOne       = `fe8a6a0c000c02fde80b0011774f01d26412f0d16ea3f0447be0b5ebec67b0782e321a7a01cbdf7f734e30fde90b02004e53753e3fe4667073063a17987292cfdea278824e9888e52180581d7188d801fdf50500262bccabec6c4af3ed00cc7a7414edea9c5efa92fb8623dd6160a001450a528201fdfb020101fd7c010093b3efca9b77ddec914f8effac691ecb54e2c81d0ab81cbc4c4b93befe418e8501bf01015e005881826eb6973c54003a02118fe270f03d46d02681c8bc71cd44c613e86302f8012e00e07a2bb8bb75e5accff266022e1e5e6e7b4d6d943a04faadcf2ab4a22f796ff30116008120cafa17309c0bb0e0ffce835286b3a2dcae48e4497ae2d2b7ced4f051507d010a00502e59ac92f46543c23006bff855d96f5e648043f0fb87a7a5949e6a9bebae430104001ccd9f8f64f4d0489b30cc815351cf425e0e78ad79a589350e4341ac165dbe45010301010000af8764ce7e1cc132ab5ed2229a005c87201c9a5ee15c0f91dd53eff31ab30cd4`
	hexTwo       = `fe8a6a0c000c02fdea0b025e441996fc53f0191d649e68a200e752fb5f39e0d5617083408fa179ddc5c998fdeb0b0101fdf405000671394f72237d08a4277f4435e5b6edf7adc272f25effef27cdfe805ce71a8101fdfb020101fd7c010093b3efca9b77ddec914f8effac691ecb54e2c81d0ab81cbc4c4b93befe418e8501bf01015e005881826eb6973c54003a02118fe270f03d46d02681c8bc71cd44c613e86302f8012e00e07a2bb8bb75e5accff266022e1e5e6e7b4d6d943a04faadcf2ab4a22f796ff30116008120cafa17309c0bb0e0ffce835286b3a2dcae48e4497ae2d2b7ced4f051507d010a00502e59ac92f46543c23006bff855d96f5e648043f0fb87a7a5949e6a9bebae430104001ccd9f8f64f4d0489b30cc815351cf425e0e78ad79a589350e4341ac165dbe45010301010000af8764ce7e1cc132ab5ed2229a005c87201c9a5ee15c0f91dd53eff31ab30cd4`
	hexCombined  = `fe8a6a0c000c04fde80b0011774f01d26412f0d16ea3f0447be0b5ebec67b0782e321a7a01cbdf7f734e30fde90b02004e53753e3fe4667073063a17987292cfdea278824e9888e52180581d7188d8fdea0b025e441996fc53f0191d649e68a200e752fb5f39e0d5617083408fa179ddc5c998fdeb0b0102fdf50500262bccabec6c4af3ed00cc7a7414edea9c5efa92fb8623dd6160a001450a5282fdf405000671394f72237d08a4277f4435e5b6edf7adc272f25effef27cdfe805ce71a8101fdfb020101fd7c010093b3efca9b77ddec914f8effac691ecb54e2c81d0ab81cbc4c4b93befe418e8501bf01015e005881826eb6973c54003a02118fe270f03d46d02681c8bc71cd44c613e86302f8012e00e07a2bb8bb75e5accff266022e1e5e6e7b4d6d943a04faadcf2ab4a22f796ff30116008120cafa17309c0bb0e0ffce835286b3a2dcae48e4497ae2d2b7ced4f051507d010a00502e59ac92f46543c23006bff855d96f5e648043f0fb87a7a5949e6a9bebae430104001ccd9f8f64f4d0489b30cc815351cf425e0e78ad79a589350e4341ac165dbe45010301010000af8764ce7e1cc132ab5ed2229a005c87201c9a5ee15c0f91dd53eff31ab30cd4`
	rootExample  = `bb6f640cc4ee56bf38eb5a1969ac0c16caa2d3d202b22bf3735d10eec0ca6e00`
	txidExample  = `3ecead27a44d013ad1aae40038acbb1883ac9242406808bb4667c15b4f164eac`
)

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
	bump, err := NewBUMPFromStr(hexExample)
	require.NoError(t, err)
	root, err := bump.CalculateRootGivenTxid(txidExample)
	require.NoError(t, err)
	require.Equal(t, rootExample, root)
}

func TestNewBUMPFromStr2(t *testing.T) {
	bump, err := NewBUMPFromStr2(hexExample)
	require.NoError(t, err)
	str, err := bump.String()
	require.NoError(t, err)
	require.Equal(t, hexExample, str)
}

func TestNewBUMPFromJson2(t *testing.T) {
	jBump, err := NewBUMPFromJSON2(jsonExample2)
	require.NoError(t, err)
	jStr, err := jBump.String()
	require.NoError(t, err)
	require.Equal(t, hexExample, jStr)
}

func TestCalculateRootGivenTxid2(t *testing.T) {
	bump, err := NewBUMPFromStr2(hexExample)
	require.NoError(t, err)
	root, err := bump.CalculateRootGivenTxid(txidExample)
	require.NoError(t, err)
	require.Equal(t, rootExample, root)
}
func TestAddingBUMPs1(t *testing.T) {
	bump, err := NewBUMPFromStr(hexOne)
	require.NoError(t, err)
	bump2, err := NewBUMPFromStr(hexTwo)
	require.NoError(t, err)
	_, err = bump.Add(bump2)
	require.NoError(t, err)
	// combined, err := c.String()
	// require.NoError(t, err)
	// require.Equal(t, hexCombined, combined)
}

func TestAddingBUMPs2(t *testing.T) {
	bump, err := NewBUMPFromStr2(hexOne)
	require.NoError(t, err)
	bump2, err := NewBUMPFromStr2(hexTwo)
	require.NoError(t, err)
	err = bump.Add(bump2)
	require.NoError(t, err)
	combined, err := bump.String()
	require.NoError(t, err)
	require.Equal(t, hexCombined, combined)
}

func BenchmarkCalculateRootOption1(b *testing.B) {
	for n := 0; n < b.N; n++ {
		bump, err := NewBUMPFromStr(hexExample)
		require.NoError(b, err)
		root, err := bump.CalculateRootGivenTxid(txidExample)
		require.NoError(b, err)
		require.Equal(b, rootExample, root)
	}
}

func BenchmarkCalculateRootOption2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		bump, err := NewBUMPFromStr2(hexExample)
		require.NoError(b, err)
		root, err := bump.CalculateRootGivenTxid(txidExample)
		require.NoError(b, err)
		require.Equal(b, rootExample, root)
	}
}

func BenchmarkAdd1(b *testing.B) {
	for n := 0; n < b.N; n++ {
		bump, err := NewBUMPFromStr(hexOne)
		require.NoError(b, err)
		bump2, err := NewBUMPFromStr(hexTwo)
		require.NoError(b, err)
		_, err = bump.Add(bump2)
		require.NoError(b, err)
		// combined, err := c.String()
		// require.NoError(b, err)
		// require.Equal(b, hexCombined, combined)
	}
}

func BenchmarkAdd2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		bump, err := NewBUMPFromStr2(hexOne)
		require.NoError(b, err)
		bump2, err := NewBUMPFromStr2(hexTwo)
		require.NoError(b, err)
		err = bump.Add(bump2)
		require.NoError(b, err)
		// combined, err := bump.String()
		// require.NoError(b, err)
		// require.Equal(b, hexCombined, combined)
	}
}
