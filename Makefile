
filtered_camera: *.go cmd/module/*.go
	go build -o filtered_camera cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .
