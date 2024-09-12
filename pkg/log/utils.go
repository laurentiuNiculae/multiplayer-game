package log

import (
	"strings"
)

func faintFilePath(p []byte) []byte {
	str := string(p)

	firstLineEndIndex := strings.Index(str, "\n")

	filePath := str[:firstLineEndIndex]

	return []byte(faint(filePath) + str[firstLineEndIndex:])
}
