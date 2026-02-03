package lineage

import (
	"fmt"
	"strings"
)

func shellQuote(value string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "'\\''"))
}
