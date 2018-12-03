FROM golang:latest as builder
ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH=/go
ENV GOBIN=/
ENV APPPATH=/github.com/xnaveira/alertdelay
RUN apt-get update && apt-get install -y gccgo
RUN go get -u github.com/golang/dep/cmd/dep
ADD . $GOPATH/src$APPPATH
WORKDIR $GOPATH/src$APPPATH
RUN $GOBIN/dep ensure && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .
FROM golang:alpine
COPY --from=builder /go/src/github.com/xnaveira/alertdelay/main /app/
WORKDIR /app
CMD ["/app/main"]