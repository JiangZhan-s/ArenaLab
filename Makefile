.PHONY: verify test-rules test-race course-check
verify:
	go vet ./...
	go test ./... -timeout=60s
	go run ./cmd/course check
test-rules:
	go test -tags=acceptance ./test/acceptance/rules -count=1 -timeout=30s
test-race:
	go test -race ./... -timeout=60s
course-check:
	go run ./cmd/course check
