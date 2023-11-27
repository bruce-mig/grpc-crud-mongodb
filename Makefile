build:
	go build -o bin/blogger

run: build
	./bin/blogger

proto:
	 protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/blog.proto

.PHONY: proto