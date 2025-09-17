.PHONY: Tages
proto:
	cd proto && protoc \
		--go_out=../pkg --go_opt=paths=source_relative \
		--go-grpc_out=../pkg --go-grpc_opt=paths=source_relative \
		service.proto
clean:
	rm -f pkg/*.pb.go
run-docker:
	docker compose -f docker-compose.yml up -d --build

down:
	docker compose -f docker-compose.yml down --volumes --remove-orphans
run-all:
	docker compose -f docker-compose.yml up -d --build
	go run ./cmd/main.go