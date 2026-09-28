deps:
	git submodule update --init --recursive
	(cd extensions && npm ci --ignore-scripts)
	(cd js && npm ci --ignore-scripts)

extensions:
	(cd extensions && npm run build)

js:
	(cd js && npm run build)

example: extensions
	CGO_ENABLED=0 go build -o ./build/example -ldflags="-s -w" ./cmd/example
	./build/example 127.0.0.1:8081

example-race: extensions
	go build -race -o ./build/example ./cmd/example
	./build/example 127.0.0.1:8081

reload:
	curl --fail-with-body -X PUT -F file=@cmd/example/$(or $(file),event/hello.js) http://127.0.0.1:8081/reload

typecheck:
	(cd extensions && npm run typecheck)
	(cd js && npm run typecheck)

test:
	go test -race ./...

.PHONY: deps js extensions example example-race reload typecheck test
