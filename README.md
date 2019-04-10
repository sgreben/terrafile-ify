# terrafile-ify

`terrafile-ify` generates Terrafiles and (optionally) re-writes Terraform source to use vendored modules.

## Contents

- [Contents](#contents)
- [Get it](#get-it)
- [Usage](#usage)

## Get it

Using go get:

```bash
go get -u github.com/sgreben/terrafile-ify
```

Or [download the binary](https://github.com/sgreben/terrafile-ify/releases/latest) from the releases page.

```bash
# Linux
curl -LO https://github.com/sgreben/terrafile-ify/releases/download/0.0.1/terrafile-ify_0.0.1_linux_x86_64.zip
unzip terrafile-ify_0.0.1_linux_x86_64.zip

# OS X
curl -LO https://github.com/sgreben/terrafile-ify/releases/download/0.0.1/terrafile-ify_0.0.1_osx_x86_64.zip
unzip terrafile-ify_0.0.1_osx_x86_64.zip

# Windows
curl -LO https://github.com/sgreben/terrafile-ify/releases/download/0.0.1/terrafile-ify_0.0.1_windows_x86_64.zip
unzip terrafile-ify_0.0.1_windows_x86_64.zip
```

## Usage

```text
Usage of terrafile-ify:
  -execute
    	run the terrafile binary on each directory (default: false)
  -generate
    	generate Terrafiles on disk (default: false, just print to stdout)
  -ignore string
    	ignore files and directories matching this glob pattern (default ".terraform")
  -rewrite
    	rewrite files in-place (default: false)
  -version
    	print version and exit
```
