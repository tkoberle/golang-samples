.PHONY: proto
proto:
	@echo "Generating proto files..."
	#@if exists "proto/pb" rd /s /q "proto/pb"
	#@mkdir proto/pb
	@protoc -I ./proto --go_out=./proto/pb --go-grpc_out=./proto ./proto/echo-service.proto
