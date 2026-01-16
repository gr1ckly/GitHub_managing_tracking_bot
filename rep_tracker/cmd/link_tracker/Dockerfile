FROM golang:1.22 AS build

WORKDIR /app

RUN apt-get update \
  && apt-get install -y --no-install-recommends protobuf-compiler \
  && rm -rf /var/lib/apt/lists/*

ENV PATH="/root/go/bin:${PATH}"
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
  && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY . .
RUN protoc --go_out=rep_tracker/pkg/proto --go-grpc_out=rep_tracker/pkg/proto proto/rep_tracker.proto

WORKDIR /app/rep_tracker
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/link_tracker ./cmd/link_tracker

FROM gcr.io/distroless/base-debian12

COPY --from=build /out/link_tracker /link_tracker

ENTRYPOINT ["/link_tracker"]
