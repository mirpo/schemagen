package parse

import (
	"fmt"
	"os"
)

func Load(path string) ([]*NamedSchema, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		return ParseDir(path)
	}

	ns, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	return []*NamedSchema{ns}, nil
}
