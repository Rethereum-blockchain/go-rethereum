print "Building Geth Linux binary"
let-env GOOS = linux; go build -trimpath -ldflags="-s -w" -o release/ ./cmd/geth
print "Building Geth Windows binary"
let-env GOOS = windows; go build -trimpath -ldflags="-s -w" -o release/ ./cmd/geth
print "Building Clef Linux binary"
let-env GOOS = linux; go build -trimpath -ldflags="-s -w" -o release/ ./cmd/clef
print "Building Clef Windows binary"
let-env GOOS = windows; go build -trimpath -ldflags="-s -w" -o release/ ./cmd/clef