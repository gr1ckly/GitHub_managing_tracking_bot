FROM golang:1.24.10 AS build

WORKDIR /app

RUN apt-get update \
  && apt-get install -y --no-install-recommends protobuf-compiler libprotobuf-dev \
  && rm -rf /var/lib/apt/lists/*

ENV PATH="/root/go/bin:${PATH}"
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
  && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY . .
RUN protoc -I proto -I /usr/include --go_out=rep_tracker/pkg/proto --go-grpc_out=rep_tracker/pkg/proto proto/rep_tracker.proto

WORKDIR /app/rep_tracker
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/rep_tracker_server ./cmd/server

FROM gcr.io/distroless/base-debian12

COPY --from=build /out/rep_tracker_server /rep_tracker_server

ENTRYPOINT ["/rep_tracker_server"]
