package export

import "strings"

func MarkdownFile(title, content string) ([]byte, string) {
	name := strings.ReplaceAll(strings.TrimSpace(title), "/", "-") + ".md"
	return []byte(content), name
}
