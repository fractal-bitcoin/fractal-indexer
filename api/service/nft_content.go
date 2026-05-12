package service

import "fractal-query/constant"

func decodeNFTContent(contentCode uint8, content string) string {
	if decoded, ok := constant.GetNFTContentByCode(contentCode); ok {
		return decoded
	}
	return content
}
