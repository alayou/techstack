package model

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/GehirnInc/crypt"
	"github.com/GehirnInc/crypt/apr1_crypt"
	"github.com/GehirnInc/crypt/md5_crypt"
	"github.com/GehirnInc/crypt/sha256_crypt"
	"github.com/GehirnInc/crypt/sha512_crypt"
	"github.com/alayou/techstack/utils"
	"github.com/alexedwards/argon2id"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

const (
	argonPwdPrefix            = "$argon2id$"
	bcryptPwdPrefix           = "$2a$"
	pbkdf2SHA1Prefix          = "$pbkdf2-sha1$"
	pbkdf2SHA256Prefix        = "$pbkdf2-sha256$"
	pbkdf2SHA512Prefix        = "$pbkdf2-sha512$"
	pbkdf2SHA256B64SaltPrefix = "$pbkdf2-b64salt-sha256$"
	md5cryptPwdPrefix         = "$1$"
	md5cryptApr1PwdPrefix     = "$apr1$"
	sha256cryptPwdPrefix      = "$5$"
	sha512cryptPwdPrefix      = "$6$"
	yescryptPwdPrefix         = "$y$"
	md5DigestPwdPrefix        = "{MD5}"
	sha256DigestPwdPrefix     = "{SHA256}"
	sha512DigestPwdPrefix     = "{SHA512}"
)

var (
	hashPwdPrefixes = []string{argonPwdPrefix, bcryptPwdPrefix, pbkdf2SHA1Prefix, pbkdf2SHA256Prefix,
		pbkdf2SHA512Prefix, pbkdf2SHA256B64SaltPrefix, md5cryptPwdPrefix, md5cryptApr1PwdPrefix, md5DigestPwdPrefix,
		sha256DigestPwdPrefix, sha512DigestPwdPrefix, sha256cryptPwdPrefix, sha512cryptPwdPrefix, yescryptPwdPrefix}
	pbkdfPwdPrefixes        = []string{pbkdf2SHA1Prefix, pbkdf2SHA256Prefix, pbkdf2SHA512Prefix, pbkdf2SHA256B64SaltPrefix}
	pbkdfPwdB64SaltPrefixes = []string{pbkdf2SHA256B64SaltPrefix}
	unixPwdPrefixes         = []string{md5cryptPwdPrefix, md5cryptApr1PwdPrefix, sha256cryptPwdPrefix, sha512cryptPwdPrefix,
		yescryptPwdPrefix}
	digestPwdPrefixes = []string{md5DigestPwdPrefix, sha256DigestPwdPrefix, sha512DigestPwdPrefix}
)

func (u *User) PasswordEqual(password string) (bool, error) {
	return PasswordEqual(u.Password, password)
}

func PasswordEqual(hashedPassword, password string) (bool, error) {
	match := false
	var err error
	switch {
	case strings.HasPrefix(hashedPassword, bcryptPwdPrefix):
		if err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
			return false, err
		}
		match = true
	case strings.HasPrefix(hashedPassword, argonPwdPrefix):
		match, err = argon2id.ComparePasswordAndHash(password, hashedPassword)
		if err != nil {
			return match, err
		}
	case utils.IsStringPrefixInSlice(hashedPassword, unixPwdPrefixes):
		match, err = compareUnixPasswordAndHash(hashedPassword, password)
		if err != nil {
			return match, err
		}
	case utils.IsStringPrefixInSlice(hashedPassword, pbkdfPwdPrefixes):
		match, err = comparePbkdf2PasswordAndHash(password, hashedPassword)
		if err != nil {
			return match, err
		}
	case utils.IsStringPrefixInSlice(hashedPassword, digestPwdPrefixes):
		match = compareDigestPasswordAndHash(hashedPassword, password)
	}

	return match, err
}

func compareDigestPasswordAndHash(hashedPassword, password string) bool {
	if strings.HasPrefix(hashedPassword, md5DigestPwdPrefix) {
		h := md5.New() // nolint
		h.Write([]byte(password))
		return fmt.Sprintf("%s%x", md5DigestPwdPrefix, h.Sum(nil)) == hashedPassword
	}
	if strings.HasPrefix(hashedPassword, sha256DigestPwdPrefix) {
		h := sha256.New()
		h.Write([]byte(password))
		return fmt.Sprintf("%s%x", sha256DigestPwdPrefix, h.Sum(nil)) == hashedPassword
	}
	if strings.HasPrefix(hashedPassword, sha512DigestPwdPrefix) {
		h := sha512.New()
		h.Write([]byte(password))
		return fmt.Sprintf("%s%x", sha512DigestPwdPrefix, h.Sum(nil)) == hashedPassword
	}
	return false
}

func compareUnixPasswordAndHash(hashedPassword, password string) (bool, error) {
	if strings.HasPrefix(hashedPassword, yescryptPwdPrefix) {
		return false, errors.New("yescrypt hash format is not supported or disabled")
	}
	var crypter crypt.Crypter
	if strings.HasPrefix(hashedPassword, sha512cryptPwdPrefix) {
		crypter = sha512_crypt.New()
	} else if strings.HasPrefix(hashedPassword, sha256cryptPwdPrefix) {
		crypter = sha256_crypt.New()
	} else if strings.HasPrefix(hashedPassword, md5cryptPwdPrefix) {
		crypter = md5_crypt.New()
	} else if strings.HasPrefix(hashedPassword, md5cryptApr1PwdPrefix) {
		crypter = apr1_crypt.New()
	} else {
		return false, errors.New("unix crypt: invalid or unsupported hash format")
	}
	if err := crypter.Verify(hashedPassword, []byte(password)); err != nil {
		return false, err
	}
	return true, nil
}

func comparePbkdf2PasswordAndHash(password, hashedPassword string) (bool, error) {
	vals := strings.Split(hashedPassword, "$")
	if len(vals) != 5 {
		return false, fmt.Errorf("pbkdf2: hash is not in the correct format")
	}
	iterations, err := strconv.Atoi(vals[2])
	if err != nil {
		return false, err
	}
	expected, err := base64.StdEncoding.DecodeString(vals[4])
	if err != nil {
		return false, err
	}
	var salt []byte
	if utils.IsStringPrefixInSlice(hashedPassword, pbkdfPwdB64SaltPrefixes) {
		salt, err = base64.StdEncoding.DecodeString(vals[3])
		if err != nil {
			return false, err
		}
	} else {
		salt = []byte(vals[3])
	}
	var hashFunc func() hash.Hash
	if strings.HasPrefix(hashedPassword, pbkdf2SHA256Prefix) || strings.HasPrefix(hashedPassword, pbkdf2SHA256B64SaltPrefix) {
		hashFunc = sha256.New
	} else if strings.HasPrefix(hashedPassword, pbkdf2SHA512Prefix) {
		hashFunc = sha512.New
	} else if strings.HasPrefix(hashedPassword, pbkdf2SHA1Prefix) {
		hashFunc = sha1.New
	} else {
		return false, fmt.Errorf("pbkdf2: invalid or unsupported hash format %v", vals[1])
	}
	df := pbkdf2.Key([]byte(password), salt, iterations, len(expected), hashFunc)
	return subtle.ConstantTimeCompare(df, expected) == 1, nil
}
