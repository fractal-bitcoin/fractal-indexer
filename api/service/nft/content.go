package events

import "fractal-query/constant"

func decodeNFTEventContent(contentCode uint8, content string) string {
	if decoded, ok := constant.GetNFTContentByCode(contentCode); ok {
		return decoded
	}
	return content
}
