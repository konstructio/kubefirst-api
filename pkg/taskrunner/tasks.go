package taskrunner

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"text/template"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

func expandPath(d, p string) ([]string, error) {
	return filepath.Glob(path.Join(d, p))
}

type Command interface {
	exec(c *Config) error
}

type TaskFile struct {
	Tasks []Task `json:"tasks"`
}

type Task struct {
	Copy *copyTask `json:"copy,omitempty"`
	Remove *removeTask `json:"remove,omitempty"`
}

func (t *Task) exec(c *Config) error {
	v := reflect.ValueOf(*t)

	// Ensure only one command set
	var command Command
	for i := 0; i < v.NumField(); i++ {
		isNil := v.Field(i).IsNil()
		if !isNil {
			if command != nil {
				return ErrConflict
			}
			command = v.Field(i).Interface().(Command)
		}
	}

	return command.exec(c)
}

type Config struct {
	Root      string
	TaskFile  string
	Variables map[string]string

	parsedTaskFile bytes.Buffer
}

func (c *Config) Exec() error {
	// Load the task file
	f, err := os.ReadFile(path.Join(c.Root, c.TaskFile))
	if err != nil {
		return err
	}

	t := template.New("tpl")
	t, err = t.Parse(string(f))
	if err != nil {
		return err
	}

	if err := t.Execute(&c.parsedTaskFile, c.Variables); err != nil {
		return err
	}

	taskFile := &TaskFile{}
	if err := yaml.Unmarshal(c.parsedTaskFile.Bytes(), taskFile); err != nil {
		return err
	}

	log.Debug().Interface("taskrunner",c.parsedTaskFile.String()).Msg("Task runner file generated")

	for _, cmd := range taskFile.Tasks {
		if err := cmd.exec(c); err != nil {
			return err
		}
	}

	return nil
}
