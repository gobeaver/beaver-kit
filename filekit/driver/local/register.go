package local

import "github.com/gobeaver/beaver-kit/filekit"

func init() {
	filekit.RegisterDriver("local", func(cfg filekit.Config) (filekit.FileSystem, error) {
		return New(cfg.LocalBasePath)
	})
}
