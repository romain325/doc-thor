package stages

import (
	"fmt"
	"os"
)

// Collect verifies that the output directory is non-empty after the build
// container exits. The actual file walk happens in Upload.
func Collect(outputDir string) error {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("read output dir: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("output directory is empty — container produced no output")
	}
	return nil
}
