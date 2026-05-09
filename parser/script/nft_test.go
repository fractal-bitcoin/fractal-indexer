package script

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
)

var nftScripts []string

func init() {
	dat, err := ioutil.ReadFile("nft.txt")
	if err != nil {
		panic(err)
	}
	nftScripts = strings.Split(string(dat), "\n")
}

func TestNFTDecode(t *testing.T) {
	for line, scriptHex := range nftScripts {
		if len(scriptHex) == 0 {
			continue
		}
		script, err := hex.DecodeString(scriptHex)
		if err != nil {
			t.Logf("ignore line: %d, %s", line, scriptHex)
			continue
		}

		if nft, hasnft := ExtractPkScriptForNFT(script); hasnft {
			data, _ := json.Marshal(nft)
			t.Logf("scriptLen: %s, nft: %s", scriptHex, strings.ReplaceAll(string(data), ",", "\n"))
		}
	}
}

func TestNFTJubileeDecode(t *testing.T) {
	for line, scriptHex := range nftScripts {
		if len(scriptHex) == 0 {
			continue
		}
		script, err := hex.DecodeString(scriptHex)
		if err != nil {
			t.Logf("ignore line: %d, %s", line, scriptHex)
			continue
		}

		for _, nft := range ExtractPkScriptForNFTJubilee(script, nil) {
			data, _ := json.Marshal(nft)
			t.Logf("scriptLen: %s, nft: %s", scriptHex, strings.ReplaceAll(string(data), ",", "\n"))
		}
	}
}
