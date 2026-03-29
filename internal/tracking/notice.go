package tracking

import (
	"fmt"
)

// HandleTrackingNotice informa o modo de tracking ativo.
func HandleTrackingNotice() {
	if !IsCgoEnabled {
		fmt.Println("Notice: Using JSON-based progress tracking (SQLite not available)")
	}
}
