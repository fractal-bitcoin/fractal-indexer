package service

import "fractal-indexer/api/constant"

func decodeNFTContent(contentCode uint8, content string) string {
	if decoded, ok := constant.GetNFTContentByCode(contentCode); ok {
		return decoded
	}
	return content
}
