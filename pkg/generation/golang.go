package generation

import (
	"github.com/mirpo/schemagen/pkg/lang/golang"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func newGoGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	return newCombinedGenerator(
		golang.NewGenerator(graph, &golang.Config{
			PackageName:      cfg.Go.PackageName,
			UsePointers:      cfg.Go.UsePointers,
			OmitEmpty:        cfg.Go.OmitEmpty,
			DisableComments:  false,
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
		}),
		&GoImportConverter{ModulePath: cfg.Go.ModulePath},
	)
}
