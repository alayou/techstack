package utils

import (
	"strings"

	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v3"
	"github.com/rs/xid"
)

// IsStringPrefixInSlice searches a string prefix in a slice and returns true
// if a matching prefix is found
func IsStringPrefixInSlice(obj string, list []string) bool {
	for i := 0; i < len(list); i++ {
		if strings.HasPrefix(obj, list[i]) {
			return true
		}
	}
	return false
}

// GenerateUniqueID retuens an unique ID
func GenerateUniqueID() string {
	u, err := uuid.NewRandom()
	if err != nil {
		return xid.New().String()
	}
	return shortuuid.DefaultEncoder.Encode(u)
}
