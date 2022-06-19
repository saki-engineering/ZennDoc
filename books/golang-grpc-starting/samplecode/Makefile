generate:
	cd src/api && protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
		--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
		hello.proto

run-server:
	cd src/cmd/server && go run .

call-client:
	cd src/cmd/client && go run .

