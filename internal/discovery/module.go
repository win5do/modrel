package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

type Module struct {
	Name       string
	Dir        string
	RelPath    string
	ModulePath string
}

type Options struct {
	Exclude []string
}

func (m Module) TagPrefix() string {
	if m.RelPath == "." {
		return ""
	}
	return filepath.ToSlash(m.RelPath) + "/"
}

func (m Module) TagFor(version string) string {
	return m.TagPrefix() + version
}

func Discover(root string, opts ...Options) ([]Module, error) {
	options := Options{}
	if len(opts) > 0 {
		options = opts[0]
	}

	var modules []Module
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && shouldSkipDir(root, path, options) {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Name() != "go.mod" {
			return nil
		}

		module, err := parseModule(root, path)
		if err != nil {
			return err
		}
		modules = append(modules, module)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})
	return modules, nil
}

func Resolve(root string, modules []Module, target string) (Module, error) {
	absTarget := target
	if !filepath.IsAbs(absTarget) {
		absTarget = filepath.Join(root, target)
	}
	absTarget, err := filepath.Abs(absTarget)
	if err != nil {
		return Module{}, err
	}

	for _, module := range modules {
		if samePath(module.Dir, absTarget) || module.Name == target || module.RelPath == target {
			return module, nil
		}
	}
	return Module{}, fmt.Errorf("no discovered Go module matches %q", target)
}

func parseModule(root, goModPath string) (Module, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return Module{}, err
	}
	file, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return Module{}, err
	}
	if file.Module == nil || file.Module.Mod.Path == "" {
		return Module{}, fmt.Errorf("%s does not declare a module path", goModPath)
	}

	dir := filepath.Dir(goModPath)
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return Module{}, err
	}
	if rel == "" {
		rel = "."
	}
	rel = filepath.ToSlash(rel)
	name := rel
	if name == "" {
		name = "."
	}

	return Module{
		Name:       name,
		Dir:        dir,
		RelPath:    rel,
		ModulePath: file.Module.Mod.Path,
	}, nil
}

func shouldSkipDir(root, path string, opts Options) bool {
	name := filepath.Base(path)
	if name == ".git" || name == "vendor" || name == "testdata" {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, ".git/") {
		return true
	}
	return matchesExclude(rel, opts.Exclude)
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return left == right
	}
	return leftAbs == rightAbs
}

func matchesExclude(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if pattern == rel {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
		}
		if ok, _ := filepath.Match(pattern, rel); ok {
			return true
		}
	}
	return false
}
