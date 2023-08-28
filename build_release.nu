print "----- Building AMD64 binaries -----"

print "Building AMD64 Linux binaries"
let-env GOOS = linux;
let-env GOARCH = amd64;
go build -v -trimpath -ldflags="-s -w" -o release/linux_amd64/ ./cmd/...

print "Building AMD64 Windows binaries"
let-env GOOS = windows;
let-env GOARCH = amd64;
go build -v -trimpath -ldflags="-s -w" -o release/windows_amd64/ ./cmd/...

print "Building AMD64 Darwin binaries"
let-env GOOS = darwin;
let-env GOARCH = amd64;
go build -v -trimpath -ldflags="-s -w" -o release/mac_amd64/ ./cmd/...

print "----- Building ARM64 binaries -----"

print "Building ARM64 Linux binaries"
let-env GOOS = linux;
let-env GOARCH = arm64;
go build -v -trimpath -ldflags="-s -w" -o release/linux_arm64/ ./cmd/...

print "Building ARM64 Darwin binaries"
let-env GOOS = darwin;
let-env GOARCH = arm64;
go build -v -trimpath -ldflags="-s -w" -o release/mac_arm64/ ./cmd/...
