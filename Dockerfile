FROM golang:1.14-alpine AS build
WORKDIR /src
RUN apk add --no-cache git musl-dev gcc build-base
ADD . /src
RUN go mod download
RUN go run novnc_generate.go
RUN go run index_generate.go
RUN go build .
RUN go test

#FROM alpine:latest
#COPY --from=build /src/easy-novnc /
EXPOSE 8080
ENTRYPOINT ["/src/easy-novnc"]
