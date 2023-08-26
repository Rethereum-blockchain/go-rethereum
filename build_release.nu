print "----- Building AMD64 binaries -----"

print "Building AMD64 Geth Linux binary"
let-env GOOS = linux;
let-env GOARCH = amd64;
go build -trimpath -ldflags="-s -w" -o release/linux_amd64/geth ./cmd/geth

print "Building AMD64 Geth Windows binary"
let-env GOOS = windows;
let-env GOARCH = amd64;
go build -trimpath -ldflags="-s -w" -o release/windows_amd64/geth.exe ./cmd/geth

print "Building AMD64 Geth Darwin binary"
let-env GOOS = darwin;
let-env GOARCH = amd64;
go build -trimpath -ldflags="-s -w" -o release/mac_amd64/geth ./cmd/geth

print "----- Building ARM64 binaries -----"

print "Building ARM64 Geth Linux binary"
let-env GOOS = linux;
let-env GOARCH = arm64;
go build -trimpath -ldflags="-s -w" -o release/linux_arm64/geth ./cmd/geth

print "Building ARM64 Geth Darwin binary"
let-env GOOS = darwin;
let-env GOARCH = arm64;
go build -trimpath -ldflags="-s -w" -o release/mac_arm64/geth ./cmd/geth
