.PHONY: build
build: test format
	GO111MODULE=on go build -tags netgo .

.PHONY: format
format:
	GO111MODULE=on go fmt ./...

.PHONY: test
test: assets
	GO111MODULE=on go test ./...

.PHONY: assets
assets:
	cd frontend && yarn run build
	GO111MODULE=on go generate -x -v ./assets/.
	GO111MODULE=on gofmt -w ./assets/assets_vfsdata.go

.PHONY: docker
docker: build
	docker build -t crochet:latest .
