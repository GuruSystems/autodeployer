PROTOCINC = -I.
PROTOCINC += -I${GOPATH}/src/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/googleapis/googleapis/third_party/protobuf/src/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/googleapis/googleapis/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/googleapis/googleapis/third_party/
PROTOCINC += -I${GOPATH}/src/golang.conradwood.net/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis

server: proto
	go install deploymonkey-server.go parsefile.go diff.go printer.go livestate.go notify.go
client: proto
	go install deploymonkey-client.go parsefile.go printer.go

all:	client server

clean:
	rm proto/*.go || echo ok

.PHONY: proto
proto:
	@echo compiling Go proto stubs
	pwd
	@protoc ${PROTOCINC} --go_out=plugins=grpc:. proto/deploymonkey.proto
	# for json gateway to compile you need the plugins:
	# 1. protoc-gen-swagger
	# 2. protoc-gen-grpc-gateway
	# they are in package github.com/grpc-ecosystem/grpc-gateway
	@protoc ${PROTOCINC} --java_out=. proto/deploymonkey.proto

