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
    go test -bench . -benchmem ./...

# Save a benchmark run to bench/<name>.txt (default: timestamped)
# Usage: just bench-save           -> bench/2026-03-29_153012.txt
#        just bench-save baseline   -> bench/baseline.txt
bench-save name="":
    #!/usr/bin/env bash
    mkdir -p bench
    if [ -z "{{name}}" ]; then
        file="bench/$(date +%Y-%m-%d_%H%M%S).txt"
    else
        file="bench/{{name}}.txt"
    fi
    go test -bench . -benchmem -count=6 ./... | tee "$file"
    echo ""
    echo "Saved to $file"

# Compare two benchmark files with benchstat
# Usage: just bench-cmp bench/old.txt bench/new.txt
bench-cmp old new:
    benchstat {{old}} {{new}}

# Compare the two most recent benchmark files in bench/
bench-cmp-latest:
    #!/usr/bin/env bash
    files=($(ls -t bench/*.txt 2>/dev/null))
    if [ ${#files[@]} -lt 2 ]; then
        echo "Need at least 2 saved benchmarks in bench/. Run 'just bench-save' first."
        exit 1
    fi
    echo "Comparing: ${files[1]} (old) vs ${files[0]} (new)"
    echo ""
    benchstat "${files[1]}" "${files[0]}"

# List saved benchmark files
bench-ls:
    @ls -lt bench/*.txt 2>/dev/null || echo "No saved benchmarks. Run 'just bench-save' first."

# Remove all saved benchmark files
bench-clean:
    rm -rf bench/

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
