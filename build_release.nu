print "----- Building AMD64 binaries -----"

print "Building AMD64 Linux binaries"
$env.GOOS = linux;
$env.GOARCH = amd64;
go build -trimpath -ldflags="-s -w" -o release/linux_amd64/ ./cmd/...

print "Building AMD64 Windows binaries"
$env.GOOS = windows;
$env.GOARCH = amd64;
go build -trimpath -ldflags="-s -w" -o release/windows_amd64/ ./cmd/...

print "Building AMD64 Darwin binaries"
$env.GOOS = darwin;
$env.GOARCH = amd64;
go build -trimpath -ldflags="-s -w" -o release/mac_amd64/ ./cmd/...

print "----- Building ARM64 binaries -----"

print "Building ARM64 Linux binaries"
$env.GOOS = linux;
$env.GOARCH = arm64;
go build -trimpath -ldflags="-s -w" -o release/linux_arm64/ ./cmd/...

print "Building ARM64 Darwin binaries"
$env.GOOS = darwin;
$env.GOARCH = arm64;
go build -trimpath -ldflags="-s -w" -o release/mac_arm64/ ./cmd/...
