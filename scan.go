package main

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
)

type scanner struct {
	root       string
	references []moduleReference
	paths      []string
}

const terraformSourceFileExt = ".tf"

func (s *scanner) ScanFile(path string) error {
	hclSource, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read terraform source %q: v", err)
	}
	var sourceFile struct{ Module []moduleReference }
	if err := hcl.Unmarshal(hclSource, &sourceFile); err != nil {
		return fmt.Errorf("process terraform source %q: v", err)
	}
	for i := range sourceFile.Module {
		sourceFile.Module[i].path = path
	}
	s.references = append(s.references, sourceFile.Module...)
	s.paths = append(s.paths, path)
	return nil
}
