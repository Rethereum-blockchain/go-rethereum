print "Building Linux binary"
let-env GOOS = linux; go build -trimpath -ldflags="-s -w" -o release/ ./cmd/geth
print "Building Windows binary"
let-env GOOS = windows; go build -trimpath -ldflags="-s -w" -o release/ ./cmd/geth