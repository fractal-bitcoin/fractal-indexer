package parallel

import (
	"fractal-indexer/constant"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"testing"
)

func TestParseBlockMetricsParallel(t *testing.T) {
	block := &model.Block{
		Height: 1,
		Txs: []model.Tx{
			{
				WitOffset:         10,
				NewNFTDataCreated: []scriptDecoder.NFTData{{}},
				TxOuts: []model.TxOut{
					{PkScript: []byte{0x6a}},
					{PkScript: []byte{0x6a, 0x01, 0x00}},
				},
			},
			{
				TxOuts: []model.TxOut{
					{PkScript: []byte{0x6a, 0x5d, 0x02, 0x02, 0x01}},
				},
			},
			{
				WitOffset: 20,
				TxIns: []model.TxIn{
					{ScriptWitness: metricTestTacitWitness()},
				},
				TxOuts: []model.TxOut{
					{PkScript: []byte{0x51}},
				},
			},
			{
				TxOuts: []model.TxOut{
					{PkScript: metricTestAlkanesScript()},
					{PkScript: metricTestAlkanesScript()},
				},
			},
		},
		ParseData: &model.ProcessBlock{},
	}

	ParseBlockMetricsParallel(block)

	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithWitness, 2)
	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithInscription, 1)
	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithOpReturn, 3)
	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithRunesRunestone, 2)
	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithRunesEtching, 1)
	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithTacit, 1)
	assertMetric(t, block.ParseData.BlockMetrics, constant.BlockMetricTxWithAlkanes, 1)
}

func assertMetric(t *testing.T, metrics map[uint32]uint32, metric, want uint32) {
	t.Helper()
	if got := metrics[metric]; got != want {
		t.Fatalf("metric %d = %d, want %d", metric, got, want)
	}
}

func metricTestTacitWitness() []byte {
	leafScript := make([]byte, 0, 48)
	leafScript = append(leafScript, 0x20)
	leafScript = append(leafScript, make([]byte, 32)...)
	leafScript = append(leafScript, 0xac, 0x00, 0x63)
	leafScript = append(leafScript, 0x05)
	leafScript = append(leafScript, []byte("TACIT")...)
	leafScript = append(leafScript, 0x01, 0x01)
	leafScript = append(leafScript, 0x01, 0x99)
	leafScript = append(leafScript, 0x68)

	witness := []byte{0x03, 0x01, 0x30, byte(len(leafScript))}
	witness = append(witness, leafScript...)
	witness = append(witness, 0x01, 0xc0)
	return witness
}

func metricTestAlkanesScript() []byte {
	return []byte{0x6a, 0x5d, 0x04, 0xff, 0x7f, 0x01, 0x00}
}
