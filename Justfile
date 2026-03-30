# Run all tests
test:
    go test ./...

# Run all tests with race detector
test-race:
    go test -race ./...

# Run all tests verbose
test-v:
    go test -v ./...

# Run all benchmarks
bench:
    go test -bench . -benchmem ./lrmap/

# Run benchmarks with count for stability
bench-stable:
    go test -bench . -benchmem -count=6 ./lrmap/

# Run coverage and generate report
coverage:
    go test -coverprofile=coverage.txt -covermode=atomic ./...
    go tool cover -func=coverage.txt

# Run coverage and open HTML report in browser
coverage-html: coverage
    go tool cover -html=coverage.txt

# Run go vet
vet:
    go vet ./...

# Run all checks (vet, race tests)
check: vet test-race
