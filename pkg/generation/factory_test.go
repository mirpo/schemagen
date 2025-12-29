package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGenerator_SupportedLanguages(t *testing.T) {
	tests := []struct {
		name     string
		language Language
		config   func() *Config
		graph    *typegraph.Graph
	}{
		{
			name:     "typescript",
			language: LanguageTypeScript,
			graph:    &typegraph.Graph{},
			config: func() *Config {
				return &Config{
					Language: LanguageTypeScript,
					TypeScript: &TypeScriptConfig{
						UnknownAny:           false,
						AdditionalProperties: false,
					},
				}
			},
		},
		{
			name:     "python",
			language: LanguagePython,
			graph:    &typegraph.Graph{},
			config: func() *Config {
				return &Config{
					Language: LanguagePython,
					Python: &PythonConfig{
						SnakeCaseField: false,
					},
				}
			},
		},
		{
			name:     "go",
			language: LanguageGo,
			graph:    &typegraph.Graph{},
			config: func() *Config {
				return &Config{
					Language: LanguageGo,
					Go: &GoConfig{
						PackageName: "test",
					},
				}
			},
		},
		{
			name:     "nil graph allowed",
			language: LanguageTypeScript,
			graph:    nil,
			config: func() *Config {
				return &Config{
					Language: LanguageTypeScript,
					TypeScript: &TypeScriptConfig{
						UnknownAny: false,
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := createGenerator(tt.graph, tt.config())

			require.NoError(t, err)
			assert.NotNil(t, gen)
		})
	}
}

func TestCreateGenerator_UnsupportedLanguages(t *testing.T) {
	tests := []struct {
		name     string
		language Language
	}{
		{"empty", ""},
		{"rust", "rust"},
		{"java", "java"},
		{"ruby", "ruby"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := createGenerator(&typegraph.Graph{}, &Config{
				Language: tt.language,
			})

			assert.Error(t, err)
			assert.Nil(t, gen)
			assert.Contains(t, err.Error(), "unsupported language")
			assert.Contains(t, err.Error(), string(tt.language))
		})
	}
}
