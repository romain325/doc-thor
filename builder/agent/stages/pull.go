package stages

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Pull clones sourceURL into repoDir. If ref is non-empty it is passed as
// --branch, which covers branches and tags. Commit SHAs require a deeper clone
// and are not supported until git clone caching is implemented.
// The resolved ref is always returned: the caller's value when one was given,
// or the name of the default branch that was actually checked out.
func Pull(sourceURL, ref, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := []string{"clone", "--depth=1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, sourceURL, repoDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %v\n%s", err, out)
	}

	if ref != "" {
		return ref, nil
	}

	// No ref was requested — detect whichever branch the remote defaulted to.
	out, err := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("detect default branch: %w", err)
	}
	return string(bytes.TrimSpace(out)), nil
}
