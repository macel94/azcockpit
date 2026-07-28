.PHONY: all build run test clean install vet race

BINARY  := azcockpit
CMD_DIR := ./cmd/azcockpit

# ── Default ───────────────────────────────────────────────────
all: build

# ── Build ─────────────────────────────────────────────────────
build:
	go build -o $(BINARY) $(CMD_DIR)

# ── Run ───────────────────────────────────────────────────────
run: build
	./$(BINARY)

# ── Test ──────────────────────────────────────────────────────
test:
	go test ./... -v -count=1

# ── Test with race detector ───────────────────────────────────
race:
	go test ./... -v -count=1 -race

# ── Vet ───────────────────────────────────────────────────────
vet:
	go vet ./...

# ── Install as ~/go/bin/azcockpit ─────────────────────────────
install:
	go install $(CMD_DIR)

# ── Clean ─────────────────────────────────────────────────────
clean:
	rm -f $(BINARY)
	go clean