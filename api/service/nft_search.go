package service

import (
	"database/sql"
	"errors"
	"fmt"
	"fractal-indexer/api/constant"
	"fractal-indexer/api/dao/clickhouse"
	mtx "fractal-indexer/api/lib/midware"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/logger"
	"fractal-indexer/api/model"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/adhocore/jsonc"
	"go.uber.org/zap"
)

var (
	countJson5                           = 0
	globalLatestTextInscriptionCount int = 0

	numbersExp = regexp.MustCompile(`^[0-9]+$`)
)

func latestTextInscriptionNumberResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret int
	err := rows.Scan(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func GetLatestTextInscriptionNumber() (number int, err error) {
	psql := fmt.Sprintf("SELECT nftnumber FROM blknft_height WHERE nfttype = 1 OR nfttype = 65 OR nfttype = 129 ORDER BY height DESC, nftidx DESC LIMIT 1")
	blkRet, err := clickhouse.ScanOne(psql, latestTextInscriptionNumberResultSRF)
	if err != nil {
		mtx.ReportError(mtx.ErrorClickhouse)
		logger.Log.Error("query nft number failed", zap.Error(err))
		return 0, err
	}
	if blkRet == nil {
		return 0, errors.New("not exist")
	}
	number = blkRet.(int)
	return number, nil
}

func inscriptionSearchResultSRF(rows *sql.Rows) (interface{}, error) {
	content := model.InscriptionContentFromDBPool.Get().(*model.InscriptionContentFromDB)
	var ret model.NFTCreatePoint

	var txId string
	var idx uint32
	var contentCode uint8
	var contentBody string
	err := rows.Scan(&ret.Height, &ret.IdxInBlock, &txId, &idx, &content.InscriptionNumber, &content.ContentType, &contentCode, &contentBody, &content.Satoshi)
	if err != nil {
		return nil, err
	}

	content.ContentBody = decodeNFTContent(contentCode, contentBody)
	content.InscriptionId = fmt.Sprintf("%si%d", utils.GetReversedStringHex(txId), idx)
	content.InscriptionNumber = content.InscriptionNumber
	content.Height = ret.Height
	content.CreateIdxKey = ret.GetCreateIdxKey()
	return content, nil
}

func GetLatestNFTCreateIdxAndHeightRange(blkStartHeight, blkEndHeight int) (nftsRsp []*model.InscriptionContentFromDB, err error) {
	whereExpr := fmt.Sprintf("height >= %d", blkStartHeight)
	if blkEndHeight != 0 {
		whereExpr = fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)
	}

	psql := fmt.Sprintf(`
SELECT height, nftidx, txid, idx, nftnumber, content_type, content_code, content, satoshi FROM blknft_height
WHERE %s AND content_len < 102400 AND (nfttype = 1 OR nfttype = 65 OR nfttype = 129)
ORDER BY height, nftidx
`, whereExpr)

	nftsRet, err := clickhouse.ScanAll(psql, inscriptionSearchResultSRF)
	if err != nil {
		mtx.ReportError(mtx.ErrorClickhouse)
		logger.Log.Error("query nft content_type count failed", zap.Error(err))
		return nil, err
	}
	if nftsRet == nil {
		return nil, errors.New("not exist")
	}
	for _, nft := range nftsRet.([]*model.InscriptionContentFromDB) {
		nftsRsp = append(nftsRsp, nft)
	}
	return
}

func GetLatestNFTCreateIdxAndHeight(blkStartHeight, blkEndHeight int) <-chan interface{} {
	out := make(chan interface{}, 1024)

	go func() {
		defer close(out)

		const sqlStr = `SELECT height, nftidx, txid, idx, nftnumber, content_type, content_code, content, satoshi FROM blknft_height
WHERE height >= %d AND height < %d AND content_len < 102400 AND (nfttype = 1 OR nfttype = 65 OR nfttype = 129)
	ORDER BY height, nftidx`
		if blkStartHeight < 0 || blkEndHeight < 0 || blkStartHeight > blkEndHeight {
			logger.Log.Warn("invalid height range", zap.Int("startHeight", blkStartHeight), zap.Int("endHeight", blkEndHeight))
			return
		}
		if blkEndHeight == 0 {
			bestHeight, err := GetBestBlockHeight()
			if err != nil {
				logger.Log.Error("get best block height failed", zap.Error(err))
				mtx.ReportError(mtx.ErrorRedis)
				return
			}
			blkEndHeight = bestHeight + 1 // Left-closed, right-open.
		}
		logger.Log.Info("get latest nft create idx and height range", zap.Int("startHeight", blkStartHeight), zap.Int("endHeight", blkEndHeight))

		const step = 1000
		for height := blkStartHeight; height < blkEndHeight; height += step {
			end := height + step
			if end > blkEndHeight {
				end = blkEndHeight
			}
			psql := fmt.Sprintf(sqlStr, height, end)
			err := clickhouse.ScanAllAsync(psql, inscriptionSearchResultSRF, out)
			if err != nil && err != sql.ErrNoRows {
				logger.Log.Error("get latest nft create idx and height failed", zap.Error(err), zap.Int("startHeight", height), zap.Int("endHeight", height+step))
				mtx.ReportError(mtx.ErrorClickhouse)
				return
			}
		}
	}()

	return out
}

type InscriptionNamePick struct {
	Proto     string `json:"p"`
	Operation string `json:"op"`

	Name string `json:"name"` // .sats

	Title string `json:"title"` // news

	BRC20Tick    string `json:"tick"` // brc20
	BRC20Max     string `json:"max"`  // brc20
	BRC20Limit   string `json:"lim"`  // brc20
	BRC20Amount  string `json:"amt"`  // brc20
	BRC20Decimal string `json:"dec"`  // brc20
}

func validateBtcDomainInscriptionName(name string) (nameFinal, suffix string, ok bool) {
	if strings.TrimSpace(name) != name {
		return "", "", false
	}
	names := strings.Fields(name)
	if len(names) == 1 {
		nameFinal = names[0]
	} else {
		return "", "", false
	}
	parts := strings.Split(nameFinal, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	suffix = strings.ToLower(parts[1])
	return nameFinal, suffix, true
}

func validateSnsInscriptionName(name string) (nameFinal, suffix string, ok bool) {
	// Doc of namespace: https://docs.satsnames.org/sats-names/sns-spec/namespaces#registration-limitations
	// Doc of name: https://docs.satsnames.org/sats-names/sns-spec/index-names#validate-names-1

	if startsWithWhitespace(name) {
		return "", "", false
	}
	name = strings.TrimSpace(name) // whitespaces defined by Unicode.
	if !utf8.ValidString(name) {
		return "", "", false
	}
	splited := strings.Split(name, ".")
	if len(splited) != 2 {
		return "", "", false
	}
	nameFinal, suffix = name, strings.ToLower(splited[1])
	if !constant.IsValidSNSDomainName(suffix) {
		return "", "", false
	}
	if containsWhitespace(nameFinal) {
		return "", "", false
	}

	return nameFinal, suffix, true
}

func ValidateInfinityAIInscriptionName(name string) (nameFinal, suffix string, ok bool) {
	/**
	1. No leading or trailing newlines, spaces, or hidden characters.
	2. Case-sensitive and must be exactly lowercase infinityai.
	3. Digits must be ASCII 0-9.
	4. No thousands separators or other separators are allowed between digits.
	5. The inscription contentType must be "text/plain;charset=utf-8".
	6. Do not index names starting with 0, such as 0001.infinityai or 0.infinityai; numbering starts from 1.
	7. Starting height.
	*/
	if !strings.HasSuffix(name, fmt.Sprintf(".%s", constant.INFINITYAI_NAME_SUFFIX)) {
		return "", "", false
	}
	if strings.HasPrefix(name, "0") {
		return "", "", false
	}
	splited := strings.Split(name, ".")
	if len(splited) != 2 {
		return "", "", false
	}
	if !numbersExp.Match([]byte(splited[0])) {
		return "", "", false
	}

	return name, constant.INFINITYAI_NAME_SUFFIX, true
}

// startsWithWhitespace checks whether a string starts with whitespace.
func startsWithWhitespace(s string) bool {
	if len(s) == 0 {
		return false
	}
	return unicode.IsSpace(rune(s[0]))
}

func containsWhitespace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func validateSnsInscription(content string) (nameFinal, suffix string, ok bool) {
	// strip utf8 BOM
	if strings.HasPrefix(content, string([]byte{0xef, 0xbb, 0xbf, '{'})) {
		content = content[3:]
	}
	if !strings.HasPrefix(content, "{") {
		return "", "", false
	}
	if !strings.HasSuffix(content, "}") {
		return "", "", false
	}

	var namePick InscriptionNamePick
	var json5 = jsonc.New()
	countJson5++
	if err := json5.Unmarshal([]byte(content), &namePick); err != nil {
		return "", "", false
	}
	// json
	if namePick.Proto == "sns" && namePick.Operation == "reg" && namePick.Name != "" {
		return validateSnsInscriptionName(namePick.Name)
	}
	return "", "", false
}
