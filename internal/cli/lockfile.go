package cli

import (
	"fmt"
	"io"

	"github.com/LynnColeArt/gpm/internal/lockfile"
	"github.com/LynnColeArt/gpm/internal/manifest"
)

func writeLockfile(projectRoot string, file *manifest.File, stdout io.Writer) error {
	lock, err := lockfile.Build(projectRoot, file)
	if err != nil {
		return fmt.Errorf("build lockfile: %w", err)
	}

	lockPath := lockfile.Path(projectRoot)
	if err := lockfile.Save(lockPath, lock); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}

	fmt.Fprintf(stdout, "wrote %s\n", lockPath)
	return nil
}
