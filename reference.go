package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

const terraformModuleSourceGitPrefix = "git::"

type gitModuleReferenceKey string

type moduleReferenceMap map[gitModuleReferenceKey]moduleReference

type moduleReference struct {
	Source  string
	Version *string
	path    string
}

type gitModuleReference struct {
	Source   string
	refValue *string
	path     string
}

func (r gitModuleReference) Vendor() moduleReference {
	moduleName := strings.TrimSuffix(filepath.Base(r.Source), ".git")
	vendoredSource := "./" + filepath.Join("vendor", "modules", moduleName)
	return moduleReference{
		Source: vendoredSource,
		path:   r.path,
	}
}

func (r gitModuleReference) TerrafileModule() module {
	Version := "master"
	if r.refValue != nil {
		Version = *r.refValue
	}
	return module{
		Source:  r.Source,
		Version: Version,
	}
}

func (r *moduleReference) Git() (*gitModuleReference, bool) {
	if !strings.HasPrefix(r.Source, terraformModuleSourceGitPrefix) {
		return nil, false
	}
	out := gitModuleReference{
		Source: strings.TrimPrefix(r.Source, terraformModuleSourceGitPrefix),
		path:   r.path,
	}
	sourceURL, err := url.Parse(out.Source)
	if err != nil {
		return &out, true
	}
	if refValue := sourceURL.Query().Get("ref"); refValue != "" {
		out.refValue = &refValue
		sourceURL.RawQuery = ""
		out.Source = sourceURL.String()
	}
	return &out, true
}

func (r moduleReference) String() string {
	return fmt.Sprintf("%q (in %s)", r.Source, r.path)
}

func (r *gitModuleReference) Key() gitModuleReferenceKey {
	if r.refValue == nil {
		return gitModuleReferenceKey(fmt.Sprintf("%s:master", r.Source))
	}
	return gitModuleReferenceKey(fmt.Sprintf("%s:%v", r.Source, r.refValue))
}

func groupReferencesByDir(references []moduleReference) map[string][]moduleReference {
	terrafileDirs := make(map[string][]moduleReference)
	for _, r := range references {
		dir := filepath.Dir(r.path)
		terrafileDirs[dir] = append(terrafileDirs[dir], r)
	}
	return terrafileDirs
}
