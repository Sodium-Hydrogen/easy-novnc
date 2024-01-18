FROM golang:1.20-alpine AS build

WORKDIR /src
RUN apk add --no-cache git
ADD . /src
RUN go mod download
RUN go run novnc_generate.go
RUN go run index_generate.go
RUN go build .

FROM alpine:latest as release
COPY --from=build /src/easy-novnc /
EXPOSE 8080
ENTRYPOINT ["/easy-novnc"]

FROM build AS testing
RUN apk add --no-cache musl-dev gcc build-base
RUN go test
