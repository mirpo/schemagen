package generation

import (
	"github.com/mirpo/schemagen/pkg/lang/golang"
)

func newGoGenerator(cfg *Config) Generator {
	return newCombinedGenerator(
		golang.NewGenerator(&golang.Config{
			PackageName:      cfg.Go.PackageName,
			UsePointers:      cfg.Go.UsePointers,
			OmitEmpty:        cfg.Go.OmitEmpty,
			DisableComments:  false,
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
		}),
		&golang.ImportConverter{ModulePath: cfg.Go.ModulePath},
	)
}
