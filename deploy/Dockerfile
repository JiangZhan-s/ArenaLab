FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -tags reference -o /out/arena-server ./cmd/arena-server && \
    CGO_ENABLED=0 go build -buildvcs=false -tags reference -o /out/settlement-worker ./cmd/settlement-worker && \
    CGO_ENABLED=0 go build -buildvcs=false -tags reference -o /out/ds-sim ./cmd/ds-sim && \
    CGO_ENABLED=0 go build -buildvcs=false -o /out/healthcheck ./cmd/healthcheck
FROM scratch
COPY --from=build /out/ /usr/local/bin/
ENV PATH=/usr/local/bin
USER 65532:65532
CMD ["arena-server"]
