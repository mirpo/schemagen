package generation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoConfig_HasModulePath(t *testing.T) {
	cfg := &GoConfig{
		PackageName: "models",
		ModulePath:  "github.com/test/project",
	}

	assert.Equal(t, "github.com/test/project", cfg.ModulePath)
}
