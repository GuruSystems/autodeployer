PROTOCINC = -I.
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/googleapis/googleapis/third_party/protobuf/src/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/googleapis/googleapis/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/googleapis/googleapis/third_party/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis

server:
	go install autodeployer-server.go random.go linereader.go starter.go
client:
	go install autodeployer-client.go

.PHONY: proto
proto:
	@echo compiling Go proto stubs
	@protoc ${PROTOCINC} --go_out=plugins=grpc:. proto/autodeployer.proto
	# for json gateway to compile you need the plugins:
	# 1. protoc-gen-swagger
	# 2. protoc-gen-grpc-gateway
	# they are in package github.com/grpc-ecosystem/grpc-gateway
	@protoc ${PROTOCINC} --java_out=. proto/autodeployer.proto

