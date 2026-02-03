package generation

import (
	"github.com/mirpo/schemagen/pkg/lang/ts"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// newTypeScriptGenerator creates a new TypeScript generator with the given config
func newTypeScriptGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	tsCfg := &ts.Config{
		DisableHeaders:       cfg.DisableHeaders,
		DisableTimestamp:     cfg.DisableTimestamp,
		UnknownAny:           cfg.TypeScript.UnknownAny,
		AdditionalProperties: cfg.TypeScript.AdditionalProperties,
		ZodCoerceDates:       cfg.TypeScript.ZodCoerceDates,
		ZodStrict:            cfg.TypeScript.ZodStrict,
	}

	// Map Zod mode
	if cfg.TypeScript.ZodOnly {
		tsCfg.ZodMode = ts.ZodModeOnly
	} else if cfg.TypeScript.Zod {
		tsCfg.ZodMode = ts.ZodModeWithInterface
	}

	return newGeneratorWrapper(
		ts.NewGeneratorWithConfig(graph, tsCfg),
		&PassthroughConverter{},
	)
}
