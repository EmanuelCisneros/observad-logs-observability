# ── Build stage ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS build

WORKDIR /src

# Cache module downloads separately from the source so a code-only change does
# not re-download every dependency.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# CGO is disabled so the result is a fully static binary that runs on a
# scratch/distroless base. Trimpath + ldflags keep the binary small.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/observad \
    ./cmd/observad

# ── Runtime stage ──────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/observad /usr/local/bin/observad

# Distroless runs as the nonroot user by default.
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/observad"]
