FROM alpine:edge as builder

RUN apk update
RUN apk add go

RUN apk add make

ENV GOPATH=/root/go

WORKDIR /workdir
ADD go.mod go.sum ./
RUN go mod download -json

ADD . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    make build

RUN --mount=type=cache,target=/root/.cache/go-build \
    make test VERBOSE=1

FROM alpine:edge
COPY --from=builder /workdir/target/sitepkg /usr/bin/sitepkg
ENTRYPOINT [ "/usr/bin/sitepkg" ]
