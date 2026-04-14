package utils

import (
	"crypto/md5" // nolint
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gorpher/gone/core"
	"github.com/gorpher/gone/cryptoutil"
)

var key = "LwaT3E7U60oX0wH9"

func Md5HexShort(str string) string {
	hbs := md5.Sum([]byte(str)) // nolint
	return hex.EncodeToString(hbs[4:12])
}

func Sha256Hash(data []byte) string {
	sum := sha256.New().Sum(data)
	return hex.EncodeToString(sum[:])
}

//func Md5Password(username, password string) string {
//	hash := md5.New() //nolint
//	hash.Write([]byte(username))
//	hash.Write([]byte("|"))
//	hash.Write([]byte(password))
//	sum := hash.Sum(nil)
//	s := hex.EncodeToString(sum[:])
//	if len(s) > 32 {
//		return s[:32]
//	}
//	return s
//}

func EncryptAccountCode(accountID, accessToken string) (string, error) {
	cbc, err := cryptoutil.EncryptByAesCBC([]byte(fmt.Sprintf("%s|%s", accountID, accessToken)), []byte(key), []byte(key))
	if err != nil {
		return "", err
	}
	return string(core.Base64URLEncode(cbc)), err
}

func DecryptAccountCode(text string) (accountID, accessToken string, err error) {
	decode, err := core.Base64URLDecode([]byte(text))
	if err != nil {
		return "", "", err
	}
	cbc, err := cryptoutil.DecryptByAesCBC(decode, []byte(key), []byte(key))
	if err != nil {
		return "", "", err
	}
	split := strings.Split(string(cbc), "|")
	return split[0], split[1], nil
}
