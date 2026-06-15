package output

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/render"
)

func GenerateBarrelContent(barrel BarrelFile, language Language) string {
	switch language {
	case LanguageTypeScript:
		return generateTypeScriptBarrel(barrel)
	case LanguagePython:
		return generatePythonBarrel(barrel)
	default:
		return ""
	}
}

func GenerateNestedBarrels(files []OutputFile, language Language) []BarrelFile {
	if GetBarrelFileName(language) == "" {
		return nil
	}

	dirFiles := groupFilesByDirectory(files)

	barrels := make([]BarrelFile, 0, len(dirFiles))

	for dir, filesInDir := range dirFiles {
		exports := make([]string, 0, len(filesInDir))

		for _, f := range filesInDir {
			base := filepath.Base(f.RelativePath)
			name := stripExtension(base)

			if name == "index" || name == "__init__" {
				continue
			}

			exports = append(exports, name)
		}

		if len(exports) == 0 {
			continue
		}

		sort.Strings(exports)

		barrelName := GetBarrelFileName(language)
		if barrelName == "" {
			continue
		}
		barrelPath := barrelName
		if dir != "" {
			barrelPath = filepath.ToSlash(filepath.Join(dir, barrelName))
		}

		barrels = append(barrels, BarrelFile{
			Path:    barrelPath,
			Exports: exports,
		})
	}

	sort.Slice(barrels, func(i, j int) bool {
		return barrels[i].Path < barrels[j].Path
	})

	return barrels
}

func generateTypeScriptBarrel(barrel BarrelFile) string {
	var b strings.Builder

	b.WriteString(render.GenerateHeader(render.HeaderConfig{CommentPrefix: render.CommentPrefixSlash, DisableTimestamp: true}))

	for _, e := range barrel.Exports {
		b.WriteString("export * from './")
		b.WriteString(e)
		b.WriteString("';\n")
	}

	return b.String()
}

func generatePythonBarrel(barrel BarrelFile) string {
	var b strings.Builder

	b.WriteString(render.GenerateHeader(render.HeaderConfig{CommentPrefix: render.CommentPrefixHash, DisableTimestamp: true}))

	for _, e := range barrel.Exports {
		b.WriteString("from .")
		b.WriteString(e)
		b.WriteString(" import *\n")
	}

	return b.String()
}

func groupFilesByDirectory(files []OutputFile) map[string][]OutputFile {
	result := make(map[string][]OutputFile)

	for _, f := range files {
		dir := filepath.ToSlash(filepath.Dir(f.RelativePath))
		if dir == "." {
			dir = ""
		}
		result[dir] = append(result[dir], f)
	}

	return result
}
