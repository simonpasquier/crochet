.PHONY: build
build: assets
	GO111MODULE=on go build -tags netgo .

.PHONY: assets
assets:
	cd frontend && yarn run build
	GO111MODULE=on go generate -x -v ./assets/.
	GO111MODULE=on gofmt -w ./assets/assets_vfsdata.go

.PHONY: docker
docker: build
	docker build -t crochet:latest .
