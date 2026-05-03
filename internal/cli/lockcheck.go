package cli

import (
	"fmt"

	"github.com/LynnColeArt/gpm/internal/lockfile"
	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
)

func requireCurrentLockfile(root project.Root, file *manifest.File) error {
	if !project.HasGoMod(root.Dir) {
		return nil
	}

	result, err := lockfile.Check(root.Dir, file)
	if err != nil {
		return fmt.Errorf("check gpm.lock: %w", err)
	}

	if !result.Exists {
		return fmt.Errorf("missing %s; run gpm install", result.LockPath)
	}

	if !result.Current() {
		return fmt.Errorf("gpm.lock is out of date: %s; run gpm install", result.Summary())
	}

	return nil
}
