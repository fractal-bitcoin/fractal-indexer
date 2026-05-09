package search_test

import (
	"fmt"
	"fractal-query/model"
	"fractal-query/service/search"
	"testing"
)

func init() {
	var nfts []*model.InscriptionContentForSearch
	n := 10000
	nfts = make([]*model.InscriptionContentForSearch, n)
	for idx := 0; idx < 10000; idx++ {
		name := fmt.Sprintf("%d.x", idx%1000)
		nfts[idx] = &model.InscriptionContentForSearch{
			Type:              "text",
			Name:              name,
			NameForSearch:     name,
			Index:             0,
			ContentType:       "",
			ContentBody:       "",
			CreateIdxKey:      "",
			InscriptionNumber: 1,
			Height:            0, // Height of NFT show in block onCreate
			Address:           "",
		}
	}
	model.GlobalInscriptions = nfts
}

func TestSearch_String(t *testing.T) {
	total, matchCount, inscriptionCreateIdxes, err := search.SearchLatestNFTToGetCreateIdx("text", "32.x", 0, 30, false, false)
	if err != nil {
		t.Errorf("got err, %v", err)
	}
	t.Logf("total %d, matchCount %d", total, matchCount)

	for _, nft := range inscriptionCreateIdxes {
		t.Logf("name %s", nft.Name)
	}

}

func BenchmarkSearch(b *testing.B) {
	for n := 0; n < b.N; n++ {
		search.SearchLatestNFTToGetCreateIdx("text", "32.x", 0, 30, false, false)
	}
}
