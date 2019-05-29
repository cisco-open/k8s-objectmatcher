
all: license fmt vet

.PHONY: license
license:
	./scripts/check-header.sh

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

test-integration:
	go test -integration -v ./...
