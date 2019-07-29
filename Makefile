.PHONY: build
build: assets
	GO111MODULE=on go build -o webhook_ui .

.PHONY: assets
assets:
	cd frontend && yarn run build
	GO111MODULE=on go generate -x -v ./assets/.
	GO111MODULE=on gofmt -w ./assets/assets_vfsdata.go
