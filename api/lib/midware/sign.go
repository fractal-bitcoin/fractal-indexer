package midware

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

func SignSha256(input, key string) string {
	keyForSign := []byte(key)
	h := hmac.New(sha256.New, keyForSign)
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func VerifyTsWithTs(ts string, expired time.Duration) bool {
	timestamp, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	t := time.Unix(timestamp, 0)
	now := time.Now().UTC()
	if t.After(now.Add(expired)) || t.Before(now.Add(-expired)) {
		return false
	}
	return true
}
