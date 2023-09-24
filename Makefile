
filtered_camera: *.go cmd/module/*.go
	go build -o filtered_camera cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .

module: filtered_camera
	tar czf module.tar.gz filtered_camera
