package taskrunner

import (
	"fmt"
	"os"

	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

type copyTask struct {
	Source      string `json:"src"`
	Destination string `json:"dest"`
}

func (e *copyTask) exec(c *Config) error {
	l := log.With().
		Str("task type", "copy").
		Str("dest", e.Destination).
		Logger()

	// Convert source to list of files in case a glob is used
	source, err := expandPath(c.Root, e.Source)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	if err := createDirIfNotExists(e.Destination, 0755); err != nil {
		return err
	}

	for _, s := range source {
		l.Debug().Str("src", s).Msg("Copying files")

		if err := cp.Copy(s, e.Destination); err != nil {
			return err
		}

	}

	return nil
}

func createDirIfNotExists(dir string, perm os.FileMode) error {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}
