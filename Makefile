deps:
	git submodule update --remote
	(cd extensions && npm install)
	(cd js && npm install)

extensions:
	(cd extensions && npx tsc)

js:
	(cd js && npm run build)

example: extensions
	CGO_ENABLED=0 go build -o ./build/example -ldflags="-s -w" ./cmd/example
	./build/example 127.0.0.1:8081

example-race: extensions
	go build -race -o ./build/example ./cmd/example
	./build/example 127.0.0.1:8081

reload:
	curl -X PUT -F file=@cmd/example/$(or $(file),next.js) http://127.0.0.1:8081/reload

.PHONY: js extensions