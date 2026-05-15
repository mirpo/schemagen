package generation

import (
	"github.com/mirpo/schemagen/pkg/lang/py"
)

func newPythonGenerator(cfg *Config) Generator {
	return newCombinedGenerator(
		py.NewGeneratorWithConfig(&py.Config{
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
			SnakeCaseField:   cfg.Python.SnakeCaseField,
			AllowExtraFields: cfg.Python.AdditionalProperties,
		}),
		&PythonImportConverter{},
	)
}
