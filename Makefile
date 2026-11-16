.PHONY: all build test bench clean web-build

all: web-build build

build:
	go build -o toolheal.exe ./cmd/toolheal

test:
	go test -v ./...

bench:
	go run ./cmd/toolheal bench --file examples/failing_invocations.jsonl --schemas examples/tool_definitions.json

web-build:
	cd web && npm run build
	powershell -Command "Copy-Item -Path 'web/dist/*' -Destination 'internal/server/web_dist/' -Recurse -Force"

clean:
	powershell -Command "if (Test-Path toolheal.exe) { Remove-Item toolheal.exe -Force }"
	powershell -Command "if (Test-Path toolheal.db) { Remove-Item toolheal.db* -Force }"
