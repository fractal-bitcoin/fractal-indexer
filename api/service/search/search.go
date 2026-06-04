package search

import (
	"fractal-indexer/api/model"
	"fractal-indexer/logger"
	"strings"

	"go.uber.org/zap"
)

// onlyValid: true returns only the first valid result; false returns all results.
func SearchLatestNFTToGetCreateIdx(category, name string, start, end int, onlyValid, onlyEqual bool) (total, matchCount int, nftsRsp []*model.InscriptionContentForSearch, err error) {
	logger.Log.Info("SearchLatestNFTToGetCreateIdx",
		zap.String(category, name),
		zap.Int("start", start),
		zap.Int("end", end),
		zap.Bool("onlyValid", onlyValid),
		zap.Bool("onlyEqual", onlyEqual),
	)

	skip := false
	if len(name) == 0 {
		skip = true
	}

	fullMatchName := name
	var searchList []*model.InscriptionContentForSearch

	if category == model.SEARCH_TYPE_FB {
		name = strings.ToLower(name)
		fullMatchName = name
		if !strings.HasSuffix(name, ".sats") {
			fullMatchName = fullMatchName + ".sats"
		}
		searchList = model.GlobalInscriptionsGroupByFBDomainCategory
	} else if category == model.SEARCH_TYPE_NUMBER {
		searchList = model.GlobalInscriptionsGroupByNumberCategory
	} else if category == model.SEARCH_TYPE_WORD {
		searchList = model.GlobalInscriptionsGroupByWordCategory
	} else if category == model.SEARCH_TYPE_INFINITYAI {
		searchList = model.GlobalInscriptionsGroupByInfinityAICategory
	} else if category == model.SEARCH_TYPE_BRC20 {
		name = strings.ToLower(name)
		fullMatchName = name
		searchList = model.GlobalInscriptionsGroupByBRC20Category
	} else if category == "text" || category == "all" {
		searchList = model.GlobalInscriptions
		for _, subCategorey := range model.GlobalInscriptionsGroupBySubTextCategoryList {
			if ok := subCategorey.Exp.Match([]byte(name)); ok {
				searchList = subCategorey.Data
				break
			}
		}
	}

	total = len(searchList) // total is not the number of matches; it is the size of the search set.
	matchCount = 0          // matchCount is the actual number of matches.
	if skip {
		return
	}

	for _, nft := range searchList {
		if onlyValid && nft.Index != 0 {
			continue
		}
		if nft.NameForSearch == fullMatchName {
			if matchCount >= start && matchCount < end {
				nftsRsp = append(nftsRsp, nft)
			}
			matchCount++
		}
	}
	if !onlyEqual {
		for _, nft := range searchList {
			if nft.NameForSearch == fullMatchName {
				continue
			}
			if onlyValid && nft.Index != 0 {
				continue
			}
			if ok := strings.Contains(nft.NameForSearch, name); ok {
				if matchCount >= start && matchCount < end {
					nftsRsp = append(nftsRsp, nft)
				}
				matchCount++
			}
		}
	}

	return
}
