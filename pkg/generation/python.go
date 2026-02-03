package generation

import (
	"github.com/mirpo/schemagen/pkg/lang/py"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func newPythonGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	return newCombinedGenerator(
		py.NewGeneratorWithConfig(graph, &py.Config{
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
			SnakeCaseField:   cfg.Python.SnakeCaseField,
			AllowExtraFields: cfg.Python.AdditionalProperties,
		}),
		&PythonImportConverter{},
	)
}
