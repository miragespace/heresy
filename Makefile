deps:
	git submodule update --remote
	(cd extensions && npm install)
	(cd js && npm install)

extensions:
	(cd extensions && npx tsc)

js:
	(cd js && npm run build)

example: extensions
	go run ./cmd/example 127.0.0.1:8081

example-race: extensions
	go run -race ./cmd/example 127.0.0.1:8081

reload:
	curl -X PUT -F file=@cmd/example/$(or $(file),next.js) http://127.0.0.1:8081/reload

.PHONY: js extensions