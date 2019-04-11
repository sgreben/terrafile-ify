package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type module struct {
	Source  string `yaml:"source"`
	Version string `yaml:"version"`
}

type terrafile struct {
	Path    string
	Modules map[string]module
}

func (t *terrafile) Execute() error {
	cmd := exec.Command(config.TerrafileBinary)
	cmd.Dir = filepath.Dir(t.Path)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func (t *terrafile) Write() error {
	b, err := yaml.Marshal(t.Modules)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(t.Path, b, 0600)
	if err != nil {
		return err
	}
	return nil
}

func terrafileForDir(dir string, references []moduleReference) (*terrafile, error) {
	out := terrafile{
		Path:    filepath.Join(dir, "Terrafile"),
		Modules: make(map[string]module),
	}
	f, err := os.Open(out.Path)
	defer f.Close()
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return nil, err
	default:
		if err := yaml.NewDecoder(f).Decode(&out); err != nil {
			return nil, err
		}
	}
	for _, r := range references {
		r, ok := r.Git()
		if !ok {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(r.Source), ".git")
		if _, ok := out.Modules[name]; ok {
			continue
		}
		out.Modules[name] = r.TerrafileModule()
	}
	return &out, nil
}
