package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverAndResolveModules(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, filepath.Join(root, "go.mod"), "example.com/root")
	writeGoMod(t, filepath.Join(root, "boot", "go.mod"), "example.com/root/boot")
	writeGoMod(t, filepath.Join(root, "database", "kafka", "go.mod"), "example.com/root/database/kafka")
	writeGoMod(t, filepath.Join(root, "pkg", "testdata", "go.mod"), "example.com/root/pkg/testdata")
	writeGoMod(t, filepath.Join(root, "vendor", "go.mod"), "example.com/vendor")

	modules, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	got := make(map[string]Module)
	for _, module := range modules {
		got[module.Name] = module
	}
	for _, name := range []string{".", "boot", "database/kafka"} {
		if _, ok := got[name]; !ok {
			t.Fatalf("missing module %q in %#v", name, modules)
		}
	}
	if _, ok := got["pkg/testdata"]; ok {
		t.Fatalf("testdata module should be excluded")
	}
	if _, ok := got["vendor"]; ok {
		t.Fatalf("vendor module should be excluded")
	}

	boot, err := Resolve(root, modules, "boot")
	if err != nil {
		t.Fatalf("Resolve boot returned error: %v", err)
	}
	if boot.TagFor("v1.2.3") != "boot/v1.2.3" {
		t.Fatalf("boot tag = %q", boot.TagFor("v1.2.3"))
	}

	rootModule, err := Resolve(root, modules, ".")
	if err != nil {
		t.Fatalf("Resolve root returned error: %v", err)
	}
	if rootModule.TagFor("v1.2.3") != "v1.2.3" {
		t.Fatalf("root tag = %q", rootModule.TagFor("v1.2.3"))
	}
}

func TestDiscoverWithConfiguredExcludes(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, filepath.Join(root, "go.mod"), "example.com/root")
	writeGoMod(t, filepath.Join(root, "third_party", "tool", "go.mod"), "example.com/root/third_party/tool")
	writeGoMod(t, filepath.Join(root, "cmd", "demo", "go.mod"), "example.com/root/cmd/demo")

	modules, err := Discover(root, Options{Excludes: []string{"third_party/**", "cmd/demo"}})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(modules) != 1 || modules[0].Name != "." {
		t.Fatalf("modules = %#v, want only root", modules)
	}
}

func writeGoMod(t *testing.T, path string, module string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	content := []byte("module " + module + "\n\ngo 1.23\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func TestMajorVersionTags(t *testing.T) {
	for _, tc := range []struct{ dir, path, version, tag string }{
		{".", "example.com/repo/v2", "v2.1.0", "v2.1.0"},
		{"v2", "example.com/repo/v2", "v2.1.0", "v2.1.0"},
		{"database/dmq/v2", "example.com/repo/database/dmq/v2", "v2.4.1", "database/dmq/v2.4.1"},
		{"database/dmq", "example.com/repo/database/dmq/v2", "v2.4.1-rc.1", "database/dmq/v2.4.1-rc.1"},
		{"pkg/v10", "example.com/repo/pkg/v10", "v10.0.0", "pkg/v10.0.0"},
		{"v2/tool", "example.com/repo/v2/tool", "v1.0.0", "v2/tool/v1.0.0"},
		{"pkg/v2", "example.com/repo/pkg/v3", "v3.0.0", "pkg/v2/v3.0.0"},
	} {
		t.Run(tc.dir+tc.path, func(t *testing.T) {
			m := Module{RelPath: tc.dir, ModulePath: tc.path}
			if got := m.TagFor(tc.version); got != tc.tag {
				t.Fatalf("tag = %s, want %s", got, tc.tag)
			}
			if err := m.CheckVersion(tc.version); err != nil {
				t.Fatal(err)
			}
			if err := m.CheckVersion(m.InitialVersion()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDiscoveryIncludes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{".", "database/redis/plugin", "database/redis/redisboot", "database/redis/legacy", "database/mysql"} {
		writeGoMod(t, filepath.Join(root, name, "go.mod"), "example.com/repo/"+name)
	}
	for _, tc := range []struct {
		name string
		opts Options
		want []string
	}{
		{"exact deep modules", Options{Includes: []string{"database/redis/plugin", "database/redis/redisboot"}}, []string{"database/redis/plugin", "database/redis/redisboot"}},
		{"recursive excludes win", Options{Includes: []string{"database/redis/**"}, Excludes: []string{"database/redis/legacy"}}, []string{"database/redis/plugin", "database/redis/redisboot"}},
		{"glob", Options{Includes: []string{"database/redis/p*"}}, []string{"database/redis/plugin"}},
		{"root", Options{Includes: []string{"."}}, []string{"."}},
		{"no matches", Options{Includes: []string{"missing"}}, nil},
		{"excluded ancestor", Options{Includes: []string{"database/redis/plugin"}, Excludes: []string{"database/redis"}}, nil},
		{"empty includes", Options{Includes: []string{}}, []string{".", "database/mysql", "database/redis/legacy", "database/redis/plugin", "database/redis/redisboot"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modules, err := Discover(root, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, m := range modules {
				got = append(got, m.RelPath)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	// An unrelated malformed module must not break an explicitly selected module.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("invalid go.mod"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(root, Options{Includes: []string{"database/redis/plugin"}}); err != nil {
		t.Fatal(err)
	}
}
