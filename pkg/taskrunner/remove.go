package taskrunner

import (
	"os"

	"github.com/rs/zerolog/log"
)

type removeTask struct {
	Source string `json:"src"`
}

func (e *removeTask) exec(c *Config) error {
	source, err := expandPath(c.Root, e.Source)
	if err != nil {
		return err
	}

	for _, s := range source {
		log.Debug().Str("src", s).Msg("Removing files")
		if err := os.RemoveAll(s); err != nil {
			return err
		}
	}

	return nil
}
