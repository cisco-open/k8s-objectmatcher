
all: license fmt vet

license:
	./scripts/check-header.sh

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...
