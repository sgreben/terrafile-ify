package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/gobwas/glob"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl/hcl/token"
)

var (
	appName                        = "terrafile-ify"
	version                        = "SNAPSHOT"
	terraformModuleSourceGitPrefix = "git::"
	terrafileVersionDefault        = "master"
)

var config struct {
	Generate            bool
	Rewrite             bool
	Execute             bool
	Ignore              string
	ignoreGlob          glob.Glob
	PrintVersionAndExit bool
}

type moduleReference struct {
	Source         string `json:"source" yaml:"source"`
	vendoredSource string
	Version        *string `json:"version,omitempty" yaml:"version,omitempty"`
	path           string
}

func (r moduleReference) String() string {
	return fmt.Sprintf("%q (in %s)", r.Source, r.path)
}

func (r *moduleReference) Key() string {
	if r.Version == nil {
		return fmt.Sprintf("%s:", r.Source)
	}
	return fmt.Sprintf("%s:%v", r.Source, r.Version)
}

type module struct {
	Source  string `yaml:"source"`
	Version string `yaml:"version"`
}

type terrafile struct {
	Path    string
	Modules map[string]module
}

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	config.Ignore = ".terraform"
	flag.BoolVar(&config.Generate, "generate", config.Generate, "generate Terrafiles on disk (default: false, just print to stdout)")
	flag.BoolVar(&config.Rewrite, "rewrite", config.Rewrite, "rewrite files in-place (default: false)")
	flag.BoolVar(&config.Execute, "execute", config.Execute, "run the terrafile binary on each directory (default: false)")
	flag.StringVar(&config.Ignore, "ignore", config.Ignore, "ignore files and directories matching this glob pattern")
	flag.BoolVar(&config.PrintVersionAndExit, "version", config.PrintVersionAndExit, "print version and exit")
	flag.Parse()
	config.ignoreGlob = glob.MustCompile(config.Ignore, filepath.Separator)
}

func walkTerraformSourceFilesIn(path string, f func(path string) error) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if config.ignoreGlob.Match(path) {
				return filepath.SkipDir
			}
			if config.ignoreGlob.Match(filepath.Base(path)) {
				return filepath.SkipDir
			}
		}
		if filepath.Ext(path) != ".tf" {
			return nil
		}
		if config.ignoreGlob.Match(path) {
			return nil
		}
		if config.ignoreGlob.Match(filepath.Base(path)) {
			return nil
		}
		return f(path)
	})
}

func main() {
	var moduleReferences []moduleReference
	sourceFilePathSet := make(map[string]bool)

	walkErr := walkTerraformSourceFilesIn(".", func(path string) error {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read terraform source %q: v", err)
		}
		references, err := referencesFromSourceBytes(b)
		if err != nil {
			return fmt.Errorf("process terraform source %q: v", err)
		}
		for i := range references {
			references[i].path = path
		}
		moduleReferences = append(moduleReferences, references...)
		sourceFilePathSet[path] = true
		return nil
	})
	if walkErr != nil {
		log.Fatal(walkErr)
	}
	moduleMap, _ := vendoredModuleMap(moduleReferences)
	terrafiles, err := terrafilesOfReferences(moduleMap, moduleReferences)
	if err != nil {
		log.Fatal(fmt.Errorf("%q: v", err))
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	yaml.NewEncoder(os.Stdout).Encode(terrafiles)
	if config.Generate {
		for _, terrafile := range terrafiles {
			b, err := yaml.Marshal(terrafile.Modules)
			if err != nil {
				log.Fatal(err)
			}
			err = ioutil.WriteFile(terrafile.Path, b, 0600)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if config.Execute {
		for _, terrafile := range terrafiles {
			cmd := exec.Command("terrafile")
			cmd.Dir = filepath.Dir(terrafile.Path)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		}
	}
	if config.Rewrite {
		for path := range sourceFilePathSet {
			info, err := os.Stat(path)
			if err != nil {
				log.Fatal(fmt.Errorf("%q: %v", path, err))
			}
			b, err := ioutil.ReadFile(path)
			if err != nil {
				log.Fatal(fmt.Errorf("read terraform source %q: %v", path, err))
			}
			astRoot, err := hcl.Parse(string(b))
			if err != nil {
				log.Fatal(fmt.Errorf("parse terraform source %q: %v", path, err))
			}
			objList, ok := astRoot.Node.(*ast.ObjectList)
			if !ok {
				log.Fatal(fmt.Errorf("unexpected top-level node type in %q: %T", path, astRoot.Node))
			}
			if err := rewriteReferencesInAST(objList, moduleMap); err != nil {
				log.Fatal(fmt.Errorf("rewrite terraform AST %q: %v", path, err))
			}
			var buf bytes.Buffer
			if err := printer.Fprint(&buf, astRoot); err != nil {
				log.Fatal(fmt.Errorf("format terraform source %q: v", err))
			}
			fmt.Fprintln(&buf)
			if err := ioutil.WriteFile(path, buf.Bytes(), info.Mode()); err != nil {
				log.Fatal(fmt.Errorf("write terraform source %q: v", err))
			}
		}
	}
}

func terrafilesOfReferences(moduleMap map[string]moduleReference, references []moduleReference) ([]terrafile, error) {
	dirs := make(map[string][]moduleReference)
	for _, r := range references {
		dir := filepath.Dir(r.path)
		dirs[dir] = append(dirs[dir], r)
	}

	var out []terrafile
	for dir, references := range dirs {
		terrafile := terrafile{
			Path:    filepath.Join(dir, "Terrafile"),
			Modules: make(map[string]module),
		}
		seen := make(map[string]bool)
		for _, r := range references {
			if seen[r.Key()] {
				continue
			}
			if _, ok := moduleMap[r.Key()]; !ok {
				continue
			}
			source, version, err := terrafileModuleOfGitSource(r.Source)
			if err != nil {
				return nil, err
			}
			name := strings.TrimSuffix(filepath.Base(source), ".git")
			terrafile.Modules[name] = module{
				Source:  source,
				Version: version,
			}
			seen[r.Key()] = true
		}
		out = append(out, terrafile)
	}
	return out, nil
}

func terrafileModuleOfGitSource(source string) (string, string, error) {
	terrafileSource := strings.TrimPrefix(source, terraformModuleSourceGitPrefix)
	terrafileVersion := terrafileVersionDefault
	terrafileSourceURL, err := url.Parse(terrafileSource)
	if err != nil {
		return "", "", fmt.Errorf("cannot parse URL %q: %v", terrafileSource, err)
	}
	terrafileSourceRef := terrafileSourceURL.Query().Get("ref")
	if terrafileSourceRef != "" {
		terrafileVersion = terrafileSourceRef
		terrafileSourceURL.RawQuery = ""
		terrafileSource = terrafileSourceURL.String()
	}
	return terrafileSource, terrafileVersion, nil
}

func vendoredModuleMap(references []moduleReference) (out map[string]moduleReference, ignored []moduleReference) {
	out = make(map[string]moduleReference)
	for _, r := range references {
		if r.Version != nil {
			ignored = append(ignored, r)
			continue
		}
		var terrafileSource string
		var err error
		switch {
		case strings.HasPrefix(r.Source, terraformModuleSourceGitPrefix):
			terrafileSource, _, err = terrafileModuleOfGitSource(r.Source)
			if err != nil {
				log.Print(err)
				ignored = append(ignored, r)
				continue
			}
		default:
			ignored = append(ignored, r)
			continue
		}
		moduleName := strings.TrimSuffix(filepath.Base(terrafileSource), ".git")
		vendoredSource := "./" + filepath.Join("vendor", "modules", moduleName)
		mapped := moduleReference{
			Source:  vendoredSource,
			Version: nil,
			path:    r.path,
		}
		out[r.Key()] = mapped
	}
	return out, ignored
}

func rewriteReferencesInAST(root *ast.ObjectList, moduleMap map[string]moduleReference) error {
	for _, obj := range root.Items {
		switch obj.Keys[0].Token.Type {
		case token.IDENT: // ok
		default:
			continue
		}
		switch obj.Keys[0].Token.Text {
		case "module": // ok
		default:
			continue
		}
		objVal, ok := obj.Val.(*ast.ObjectType)
		if !ok {
			continue
		}
		var sourceLiteral *ast.LiteralType
		var versionObjIndex *int
		var source string
		var version *string
		for i, obj := range objVal.List.Items {
			switch obj.Keys[0].Token.Type {
			case token.IDENT: // ok
			default:
				continue
			}
			literal, ok := obj.Val.(*ast.LiteralType)
			if !ok {
				continue
			}
			value, ok := literal.Token.Value().(string)
			if !ok {
				continue
			}
			switch obj.Keys[0].Token.Text {
			case "source": // ok
				sourceLiteral = literal
				source = value
			case "version": // ok
				i := i
				versionObjIndex = &i
				version = &value
			default:
				continue
			}
		}
		reference := moduleReference{Source: source, Version: version}
		if mapped, ok := moduleMap[reference.Key()]; ok {
			sourceLiteral.Token.Text = fmt.Sprintf("%q", mapped.Source)
			if versionObjIndex != nil {
				i := *versionObjIndex
				objVal.List.Items = append(objVal.List.Items[:i], objVal.List.Items[i+1:]...)
			}
		}
	}
	return nil
}

func referencesFromSourceBytes(hclSource []byte) ([]moduleReference, error) {
	var sourceFile struct {
		Module []moduleReference
	}
	if err := hcl.Unmarshal(hclSource, &sourceFile); err != nil {
		return nil, err
	}
	return sourceFile.Module, nil
}
