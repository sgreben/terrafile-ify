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
curl -LO https://github.com/sgreben/terrafile-ify/releases/download/0.1.1/terrafile-ify_0.1.1_linux_x86_64.zip
unzip terrafile-ify_0.1.1_linux_x86_64.zip

# OS X
curl -LO https://github.com/sgreben/terrafile-ify/releases/download/0.1.1/terrafile-ify_0.1.1_osx_x86_64.zip
unzip terrafile-ify_0.1.1_osx_x86_64.zip

# Windows
curl -LO https://github.com/sgreben/terrafile-ify/releases/download/0.1.1/terrafile-ify_0.1.1_windows_x86_64.zip
unzip terrafile-ify_0.1.1_windows_x86_64.zip
```

## Usage

```text
terrafile-ify (generate|rewrite|execute)

    generate    Generate Terrafiles
    rewrite     Rewrite module references to use vendored modules
    execute     Run the `terrafile` binary on each Terrafile

Usage of terrafile-ify:
  -ignore string
    	ignore files and directories matching this glob pattern (default ".terraform")
  -terrafile-binary string
    	terrafile binary name (default "terrafile")
  -version
    	print version and exit
```
