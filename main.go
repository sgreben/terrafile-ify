package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gobwas/glob"
)

var (
	appName                 = "terrafile-ify"
	version                 = "SNAPSHOT"
	terrafileVersionDefault = "master"
)

var config struct {
	Generate            bool
	Rewrite             bool
	Execute             bool
	Ignore              string
	ignoreGlob          glob.Glob
	TerrafileBinary     string
	PrintVersionAndExit bool
}

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	config.Ignore = ".terraform"
	config.TerrafileBinary = "terrafile"
	flag.StringVar(&config.Ignore, "ignore", config.Ignore, "ignore files and directories matching this glob pattern")
	flag.StringVar(&config.TerrafileBinary, "terrafile-binary", config.TerrafileBinary, "terrafile binary name")
	flag.BoolVar(&config.PrintVersionAndExit, "version", config.PrintVersionAndExit, "print version and exit")
	flag.Parse()

	if config.PrintVersionAndExit {
		fmt.Println(version)
		os.Exit(0)
	}

	config.ignoreGlob = glob.MustCompile(config.Ignore, filepath.Separator)
	if flag.NArg() != 1 {
		log.Printf("one of the commands [generate execute rewrite] must be given")
		flag.Usage()
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "generate":
		config.Generate = true
	case "execute":
		config.Execute = true
	case "rewrite":
		config.Rewrite = true
	}
}

func main() {
	var scanner scanner
	if err := forEachTerraformSourceFileIn(".", scanner.ScanFile); err != nil {
		log.Fatal(err)
	}
	referencesForDir := groupReferencesByDir(scanner.references)
	var terrafiles []terrafile
	for dir, references := range referencesForDir {
		terrafile, err := terrafileForDir(dir, references)
		if err != nil {
			log.Fatal(fmt.Errorf("%q: v", err))
		}
		terrafiles = append(terrafiles, *terrafile)
	}

	if config.Generate {
		for _, terrafile := range terrafiles {
			log.Printf("generating %q (%d module(s))", terrafile.Path, len(terrafile.Modules))
			terrafile.Write()
		}
	}

	if config.Execute {
		for _, terrafile := range terrafiles {
			log.Printf("executing [terrafile %q]", terrafile.Path)
			if err := terrafile.Execute(); err != nil {
				log.Fatal(err)
			}
		}
	}

	if config.Rewrite {
		var rewriter rewriter
		replaceByVendored := make(moduleReferenceMap)
		for _, r := range scanner.references {
			if r, ok := r.Git(); ok {
				replaceByVendored[r.Key()] = r.Vendor()
			}
		}
		rewriter.replaceBy = replaceByVendored
		for _, path := range scanner.paths {
			log.Printf("rewriting %q", path)
			if err := rewriter.RewriteFile(path); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func forEachTerraformSourceFileIn(path string, f func(path string) error) error {
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
		if filepath.Ext(path) != terraformSourceFileExt {
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
