
filtered-camera: *.go cmd/module/*.go
	go build -o filtered-camera cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .

module: filtered-camera
	tar czf module.tar.gz filtered-camera
