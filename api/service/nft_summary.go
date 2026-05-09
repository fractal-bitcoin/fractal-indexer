package service

import (
	"database/sql"
	"errors"
	"fmt"
	"fractal-query/dao/clickhouse"
	"fractal-query/logger"
	"fractal-query/model"
	"strings"

	"go.uber.org/zap"
)

func inscriptionContentResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.NFTCreatePoint
	err := rows.Scan(&ret.Height, &ret.IdxInBlock, &ret.ContentType, &ret.Content)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func GetNFTContentByCreateIdx(blkHeight, nftIdx int) (contentType, content string, err error) {
	psql := fmt.Sprintf(`
SELECT height, nftidx, content_type, content FROM blknft_height
WHERE height = %d AND nftidx = %d
LIMIT 1
`, blkHeight, nftIdx)

	nftRet, err := clickhouse.ScanOne(psql, inscriptionContentResultSRF)
	if err != nil {
		logger.Log.Error("query nft content_type count failed", zap.Error(err))
		return "", "", err
	}
	if nftRet == nil {
		return "", "", errors.New("not exist")
	}
	nft := nftRet.(*model.NFTCreatePoint)
	return string(nft.ContentType), string(nft.Content), nil
}

func GetLatestNFTCreateIdxAndContentBySummaryTypeAndHeightRange(blkStartHeight, blkEndHeight, size int) (nftsRsp []uint64, contentsRsp []string, err error) {
	whereExpr := fmt.Sprintf("height >= %d", blkStartHeight)
	if blkEndHeight != 0 {
		whereExpr = fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)
	}

	psql := fmt.Sprintf(`
SELECT height, nftidx, '', content FROM blknft_height
WHERE %s AND nfttype != 0 AND nfttype != 64 AND nfttype != 128
ORDER BY height DESC, nftidx DESC
LIMIT %d
`, whereExpr, size)

	nftsRet, err := clickhouse.ScanAll(psql, inscriptionContentResultSRF)
	if err != nil {
		logger.Log.Error("query nft content_type count failed", zap.Error(err))
		return nil, nil, err
	}
	if nftsRet == nil {
		return nil, nil, errors.New("not exist")
	}
	for _, nft := range nftsRet.([]*model.NFTCreatePoint) {
		nftsRsp = append(nftsRsp, nft.GetCreateIdxKey())

		if len(nft.Content) > 1024 {
			contentsRsp = append(contentsRsp, string(nft.Content[:1024]))
		} else {
			contentsRsp = append(contentsRsp, string(nft.Content))
		}
	}
	return
}

func inscriptionResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.NFTCreatePoint
	err := rows.Scan(&ret.Height, &ret.IdxInBlock)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func inscriptionsStatusResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.InscriptionNumberRangeDO
	err := rows.Scan(&ret.MaxNumber, &ret.MinNumber)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func GetLatestNFTCreateIdxBySummaryTypeAndHeightRange(summary string, blkStartHeight, blkEndHeight, size int) (nftsRsp []uint64, err error) {
	whereExpr := fmt.Sprintf("height >= %d", blkStartHeight)
	if blkEndHeight != 0 {
		whereExpr = fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)
	}

	var contentTypesMatch []string
	var contentTypesNotMatch []string
	for contentType, summaryType := range summaryTypeOfContentType {
		if summaryType == summary {
			contentTypesMatch = append(contentTypesMatch, contentType)
		} else {
			contentTypesNotMatch = append(contentTypesNotMatch, contentType)
		}
	}
	var contentTypeExpr string
	if summary != "others" {
		contentTypeExpr = fmt.Sprintf("startsWith(content_type, '%s')", strings.Join(contentTypesMatch, "') OR startsWith(content_type, '"))
	} else {
		contentTypeExpr = fmt.Sprintf("NOT (startsWith(content_type, '%s'))", strings.Join(contentTypesNotMatch, "') OR startsWith(content_type, '"))
	}
	psql := fmt.Sprintf(`
SELECT height, nftidx FROM blknft_height
WHERE %s AND (nfttype = 0 OR nfttype = 64 OR nfttype = 128) AND %s
ORDER BY height DESC, nftidx DESC
LIMIT %d
`, whereExpr, contentTypeExpr, size)

	nftsRet, err := clickhouse.ScanAll(psql, inscriptionResultSRF)
	if err != nil {
		logger.Log.Error("query nft content_type count failed", zap.Error(err))
		return nil, err
	}
	if nftsRet == nil {
		return nil, errors.New("not exist")
	}
	for _, nft := range nftsRet.([]*model.NFTCreatePoint) {
		nftsRsp = append(nftsRsp, nft.GetCreateIdxKey())
	}
	return
}

func inscriptionCountResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.InscriptionCountDO
	err := rows.Scan(&ret.ContentType, &ret.Count)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func GetInscriptionsSummaryByHeightRange(blkStartHeight, blkEndHeight int) (typesCountRsp []*model.InscriptionCountByContentTypeResp, err error) {
	whereExpr := fmt.Sprintf("height >= %d", blkStartHeight)
	if blkEndHeight != 0 {
		whereExpr = fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)
	}

	psql := fmt.Sprintf(`
SELECT content_type, count(1) AS n FROM blknft_height
WHERE %s
GROUP BY content_type
`, whereExpr)

	typesCountRet, err := clickhouse.ScanAll(psql, inscriptionCountResultSRF)
	if err != nil {
		logger.Log.Error("query nft content_type count failed", zap.Error(err))
		return nil, err
	}
	if typesCountRet == nil {
		return nil, errors.New("not exist")
	}
	for _, typesCount := range typesCountRet.([]*model.InscriptionCountDO) {
		typesCountRsp = append(typesCountRsp, &model.InscriptionCountByContentTypeResp{
			ContentType: string(typesCount.ContentType),
			Count:       int(typesCount.Count),
		})
	}

	return

}

// type/subtype;charset=...
var summaryTypeOfContentType map[string]string = map[string]string{
	"application/java-archive":      "others",
	"application/EDI-X12":           "others",
	"application/EDIFACT":           "others",
	"application/javascript":        "others",
	"application/octet-stream":      "stream",
	"application/ogg":               "others",
	"application/pdf":               "others",
	"application/xhtml+xml":         "others",
	"application/x-shockwave-flash": "others",
	"application/json":              "text",
	"application/ld+json":           "others",
	"application/xml":               "others",
	"application/zip":               "others",

	"audio/mpeg":             "audio",
	"audio/x-ms-wma":         "audio",
	"audio/vnd.rn-realaudio": "audio",
	"audio/x-wav":            "audio",

	"image/gif":      "image",
	"image/jpeg":     "image",
	"image/png":      "image",
	"image/tiff":     "image",
	"image/x-icon":   "image",
	"image/vnd.djvu": "image",
	"image/svg+xml":  "image",

	"text/css":        "others",
	"text/csv":        "others",
	"text/html":       "html",
	"text/javascript": "others",
	"text/plain":      "text",
	"text/xml":        "others",

	"video/mpeg":      "video",
	"video/mp4":       "video",
	"video/quicktime": "video",
	"video/x-ms-wmv":  "video",
	"video/x-msvideo": "video",
	"video/x-flv":     "video",
	"video/webm":      "video",

	"image":       "image",
	"video":       "video",
	"audio":       "audio",
	"application": "others",
}

// type/subtype;charset=...
var summaryTypeOfContentTypeOrigin map[string]string = map[string]string{
	"application/java-archive":      "application",
	"application/EDI-X12":           "application",
	"application/EDIFACT":           "application",
	"application/javascript":        "application",
	"application/octet-stream":      "application",
	"application/ogg":               "audio",
	"application/pdf":               "pdf",
	"application/xhtml+xml":         "html",
	"application/x-shockwave-flash": "flash",
	"application/json":              "json",
	"application/ld+json":           "json",
	"application/xml":               "html",
	"application/zip":               "zip",

	"audio/mpeg":             "audio",
	"audio/x-ms-wma":         "audio",
	"audio/vnd.rn-realaudio": "audio",
	"audio/x-wav":            "audio",

	"image/gif":      "image",
	"image/jpeg":     "image",
	"image/png":      "image",
	"image/tiff":     "image",
	"image/x-icon":   "image",
	"image/vnd.djvu": "image",
	"image/svg+xml":  "image",

	"text/css":        "html",
	"text/csv":        "text",
	"text/html":       "html",
	"text/javascript": "text",
	"text/plain":      "text",
	"text/xml":        "html",

	"video/mpeg":      "video",
	"video/mp4":       "video",
	"video/quicktime": "video",
	"video/x-ms-wmv":  "video",
	"video/x-msvideo": "video",
	"video/x-flv":     "video",
	"video/webm":      "video",

	"image":       "image",
	"text":        "text",
	"video":       "video",
	"audio":       "audio",
	"application": "application",
}

func GetSummaryTypeByContentType(contentType string) (summaryType string) {
	if summaryType, ok := summaryTypeOfContentType[contentType]; ok {
		return summaryType
	}
	subType := strings.Split(contentType, ";")[0]
	if summaryType, ok := summaryTypeOfContentType[subType]; ok {
		return summaryType
	}
	mainType := strings.Split(subType, "/")[0]
	if summaryType, ok := summaryTypeOfContentType[mainType]; ok {
		return summaryType
	}
	return "others"
}
