package taskrunner_test

import (
	"archive/zip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konstructio/kubefirst-api/pkg/taskrunner"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestExec(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	dir, err := os.Getwd()
	assert.NoError(t, err)

	tests := []struct {
		Name     string
		Config   taskrunner.Config
		Error    error
		Checksum string
	}{
		{
			Name: "Civo GitHub",
			Config: taskrunner.Config{
				Root:     path.Join(dir, "testdata/default"),
				TaskFile: "taskrunner.yaml",
				Variables: map[string]string{
					"cloudProvider": "civo",
					"vcsProvider":   "github",
				},
			},
			Checksum: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			Name: "Conflicted taskrunner",
			Config: taskrunner.Config{
				Root:     path.Join(dir, "testdata/conflicted"),
				TaskFile: "taskrunner.yaml",
			},
			Error: taskrunner.ErrConflict,
		},
		{
			Name: "Missing root dir",
			Config: taskrunner.Config{
				Root:     path.Join(dir, "testdata/missing"),
				TaskFile: "taskrunner.yaml",
			},
			Error: fs.ErrNotExist,
		},
		{
			Name: "Missing taskfile",
			Config: taskrunner.Config{
				Root:     path.Join(dir, "testdata/default"),
				TaskFile: "missing.yaml",
			},
			Error: fs.ErrNotExist,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert := assert.New(t)

			outputDir, err := os.MkdirTemp(os.TempDir(), "taskrunner")
			assert.NoError(err)

			// Set the output dir in the variables
			if test.Config.Variables != nil {
				test.Config.Variables["outputDir"] = outputDir
			}

			err = test.Config.Exec()
			assert.ErrorIs(err, test.Error)

			// Compress the output and get the checksum
			if test.Error == nil {
				archive, err := os.CreateTemp(os.TempDir(), "archive-*.zip")
				assert.NoError(err)
				defer archive.Close()

				zipWriter := zip.NewWriter(archive)
				defer zipWriter.Close()

				// Zip the contents
				err = filepath.Walk(outputDir, func(fpath string, info fs.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() {
						return nil
					}

					relativeFpath := strings.Replace(fpath, outputDir, "", 1)

					file, err := os.Open(fpath)
					if err != nil {
						return err
					}
					defer file.Close()

					f, err := zipWriter.Create(relativeFpath)
					if err != nil {
						return err
					}

					_, err = io.Copy(f, file)
					if err != nil {
						return err
					}

					return nil
				})
				assert.NoError(err)

				ch := sha1.New()
				_, err = io.Copy(ch, archive)
				assert.NoError(err)

				// Ensure the checksum matches
				assert.Equal(test.Checksum, fmt.Sprintf("%x", ch.Sum(nil)))
			}
		})
	}
}
