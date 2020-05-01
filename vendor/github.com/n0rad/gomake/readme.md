# Gomake

Simple go project builder tool written in go.

The idea of `gomake` is to have a fully self contain project build system so even build tools are reproductible.
To achieve that, you have to `vendor` in your project gomake sources.

All tools used to `build`, `test` and `check` quality will be save as a lib dependency of your project (not included in your app),
so your project is fully standalone.


## Usage 

Create for example a `hack/gomake.go` file in your project:

```go
package main

import "github.com/n0rad/gomake"

func main() {
	gomake.ProjectBuilder().
		WithStep(&gomake.StepBuild{
			BinaryName: "my-app",
		}).
		MustBuild().MustExecute()
}
```

In root directory simplify calling gomake with a `Makefile` or a shell `script`

Makefile:
```makefile
.DEFAULT_GOAL := all
GOMAKE_PATH := ./hack

all:
	go run $(GOMAKE_PATH)

clean:
	go run $(GOMAKE_PATH) clean

build:
	go run $(GOMAKE_PATH) build

test:
	go run $(GOMAKE_PATH) test

quality:
	go run $(GOMAKE_PATH) quality

release:
	go run $(GOMAKE_PATH) release
```

`gomake` shell script:
```shell script
#!/bin/sh
exec go run "$( cd "$(dirname "$0")" ; pwd -P )/hack" $@
``` 

gomake uses `dist/` directory to build and `dist-tools/` directory to to build tools, those directories should be added to `.gitignore`
```gitignore
/dist/
/dist-tools/
```