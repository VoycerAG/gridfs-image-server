package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GetRandomFilename returns a random generated string with the appended extension
func GetRandomFilename(extension string) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s", time.Now().Nanosecond())))
	md := hash.Sum(nil)
	mdStr := hex.EncodeToString(md)

	return fmt.Sprintf("%s.%s", mdStr, extension)
}
