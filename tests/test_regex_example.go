package regex_test

import (
	"fmt"
	"regexp"
)

func main() {
	taskRegexp := regexp.MustCompile(`(?m)^(\d+)\.\s+\[([\w\s]+)\]\s+(.+)$`)
	testStr := `
1. [TODO] First task
2. [DONE] Second task
3. [IN PROGRESS] Third task
`
	matches := taskRegexp.FindAllStringSubmatch(testStr, -1)
	for i, match := range matches {
		fmt.Printf("Match %d: %v\n", i, match)
	}
}