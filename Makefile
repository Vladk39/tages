.PHONY: proto
proto:
	cd proto && protoc \
		--go_out=../pkg --go_opt=paths=source_relative \
		--go-grpc_out=../pkg --go-grpc_opt=paths=source_relative \
		service.proto
clean:
	rm -f pkg/*.pb.go