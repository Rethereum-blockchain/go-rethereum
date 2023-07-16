// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"encoding/binary"
	"golang.org/x/crypto/sha3"
	"hash"
	"lukechampine.com/blake3"
	"math/big"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/log"
)

const (
	datasetInitBytes   = 1 << 29 // Bytes in dataset at genesis
	datasetGrowthBytes = 1 << 22 // Dataset growth per epoch
	cacheInitBytes     = 1 << 23 // Bytes in cache at genesis
	cacheGrowthBytes   = 1 << 17 // Cache growth per epoch
	epochLength        = 32203   // Blocks per epoch
	mixBytes           = 128     // Width of mix
	hashBytes          = 64      // Hash length in bytes
	hashWords          = 16      // Number of 32 bit ints in a hash
	datasetParents     = 256     // Number of parents of each dataset element
	cacheRounds        = 3       // Number of rounds in cache production
	loopAccesses       = 64      // Number of accesses in hashimoto loop
)

// cacheSize returns the size of the ethash verification cache that belongs to a certain
// block number.
func cacheSize(block uint64) uint64 {
	epoch := int(block / epochLength)
	return calcCacheSize(epoch)
}

// calcCacheSize calculates the cache size for epoch. The cache size grows linearly,
// however, we always take the highest prime below the linearly growing threshold in order
// to reduce the risk of accidental regularities leading to cyclic behavior.
func calcCacheSize(epoch int) uint64 {
	size := cacheInitBytes + cacheGrowthBytes*uint64(epoch) - hashBytes
	for !new(big.Int).SetUint64(size / hashBytes).ProbablyPrime(1) { // Always accurate for n < 2^64
		size -= 2 * hashBytes
	}
	return size
}

// datasetSize returns the size of the ethash mining dataset that belongs to a certain
// block number.
func datasetSize(block uint64) uint64 {
	epoch := int(block / epochLength)
	return calcDatasetSize(epoch)
}

// calcDatasetSize calculates the dataset size for epoch. The dataset size grows linearly,
// however, we always take the highest prime below the linearly growing threshold in order
// to reduce the risk of accidental regularities leading to cyclic behavior.
func calcDatasetSize(epoch int) uint64 {
	size := datasetInitBytes + datasetGrowthBytes*uint64(epoch) - mixBytes
	for !new(big.Int).SetUint64(size / mixBytes).ProbablyPrime(1) { // Always accurate for n < 2^64
		size -= 2 * mixBytes
	}
	return size
}

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
type hasher func(dest []byte, data []byte)

// makeHasher creates a repetitive hasher, allowing the same hash data structures to
// be reused between hash runs instead of requiring new ones to be created. The returned
// function is not thread safe!
func makeHasher(h hash.Hash) hasher {
	if reflect.TypeOf(h).String() == "*blake3.Hasher" {
		type readerHash interface {
			hash.Hash
			XOF() *blake3.OutputReader
		}
		rh, ok := h.(readerHash)
		if !ok {
			panic("can't find Read method on hash")
		}
		outputLen := rh.Size()
		return func(dest []byte, data []byte) {
			rh.Reset()
			rh.Write(data)
			rh.XOF().Read(dest[:outputLen])
		}
	} else {
		type readerHash interface {
			hash.Hash
			Read([]byte) (int, error)
		}
		rh, ok := h.(readerHash)
		if !ok {
			panic("can't find Read method on hash")
		}
		outputLen := rh.Size()
		return func(dest []byte, data []byte) {
			rh.Reset()
			rh.Write(data)
			rh.Read(dest[:outputLen])
		}
	}
}

// seedHash is the seed to use for generating a verification cache and the mining
// dataset.
func seedHash(block uint64) []byte {
	seed := make([]byte, 32)
	if block < epochLength {
		return seed
	}
	hasherCB := makeHasher(blake3.New(32, nil))
	for i := 0; i < int(block/epochLength); i++ {
		hasherCB(seed, seed)
	}
	return seed
}

// generateCache creates a verification cache of a given size for an input seed.
// The cache production process involves first sequentially filling up 32 MB of
// memory, then performing two passes of Sergio Demian Lerner's RandMemoHash
// algorithm from Strict Memory Hard Hashing Functions (2014). The output is a
// set of 524288 64-byte values.
// This method places the result into dest in machine byte order.
func generateCache(dest []uint32, epoch uint64, seed []byte) {
	// Print some debug logs to allow analysis on low end devices
	logger := log.New("epoch", epoch)

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)

		logFn := logger.Debug
		if elapsed > 3*time.Second {
			logFn = logger.Info
		}
		logFn("Generated ethash verification cache", "elapsed", common.PrettyDuration(elapsed))
	}()
	// Convert our destination slice to a byte buffer
	var cache []byte
	cacheHdr := (*reflect.SliceHeader)(unsafe.Pointer(&cache))
	dstHdr := (*reflect.SliceHeader)(unsafe.Pointer(&dest))
	cacheHdr.Data = dstHdr.Data
	cacheHdr.Len = dstHdr.Len * 4
	cacheHdr.Cap = dstHdr.Cap * 4

	// Calculate the number of theoretical rows (we'll store in one buffer nonetheless)
	size := uint64(len(cache))
	rows := int(size) / hashBytes

	// Start a monitoring goroutine to report progress on low end devices
	var progress atomic.Uint32

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(3 * time.Second):
				logger.Info("Generating ethash verification cache", "percentage", progress.Load()*100/uint32(rows)/(cacheRounds+1), "elapsed", common.PrettyDuration(time.Since(start)))
			}
		}
	}()
	// Create a hasher to reuse between invocations
	hasherCB := makeHasher(sha3.NewLegacyKeccak512())
	//hasherCB := makeHasher(hasherCB.New(64, nil))

	// Sequentially produce the initial dataset
	hasherCB(cache, seed)
	for offset := uint64(hashBytes); offset < size; offset += hashBytes {
		hasherCB(cache[offset:], cache[offset-hashBytes:offset])
		progress.Add(1)
	}
	// Use a low-round version of randmemohash
	temp := make([]byte, hashBytes)

	for i := 0; i < cacheRounds; i++ {
		for j := 0; j < rows; j++ {
			var (
				srcOff = ((j - 1 + rows) % rows) * hashBytes
				dstOff = j * hashBytes
				xorOff = (binary.LittleEndian.Uint32(cache[dstOff:]) % uint32(rows)) * hashBytes
			)
			bitutil.XORBytes(temp, cache[srcOff:srcOff+hashBytes], cache[xorOff:xorOff+hashBytes])
			hasherCB(cache[dstOff:], temp)

			progress.Add(1)
		}
	}
	// Swap the byte order on big endian systems and return
	if !isLittleEndian() {
		swap(cache)
	}
}

// swap changes the byte order of the buffer assuming a uint32 representation.
func swap(buffer []byte) {
	for i := 0; i < len(buffer); i += 4 {
		binary.BigEndian.PutUint32(buffer[i:], binary.LittleEndian.Uint32(buffer[i:]))
	}
}

// fnv is an algorithm inspired by the FNV hash, which in some cases is used as
// a non-associative substitute for XOR. Note that we multiply the prime with
// the full 32-bit input, in contrast with the FNV-1 spec which multiplies the
// prime with one byte (octet) in turn.
func fnv(a, b uint32) uint32 {
	return a*0x01000193 ^ b
}

// fnvHash mixes in data into mix using the ethash fnv method.
func fnvHash(mix []uint32, data []uint32) {
	for i := 0; i < len(mix); i++ {
		mix[i] = mix[i]*0x01000193 ^ data[i]
	}
}

// generateDatasetItem combines data from 256 pseudorandomly selected cache nodes,
// and hashes that to compute a single dataset node.
func generateDatasetItem(cache []uint32, index uint32, hasher hasher) []byte {
	// Calculate the number of theoretical rows (we use one buffer nonetheless)
	rows := uint32(len(cache) / hashWords)

	// Initialize the mix
	mix := make([]byte, hashBytes)

	binary.LittleEndian.PutUint32(mix, cache[(index%rows)*hashWords]^index)
	for i := 1; i < hashWords; i++ {
		binary.LittleEndian.PutUint32(mix[i*4:], cache[(index%rows)*hashWords+uint32(i)])
	}
	hasher(mix, mix)

	// Convert the mix to uint32s to avoid constant bit shifting
	intMix := make([]uint32, hashWords)
	for i := 0; i < len(intMix); i++ {
		intMix[i] = binary.LittleEndian.Uint32(mix[i*4:])
	}
	// fnv it with a lot of random cache nodes based on index
	for i := uint32(0); i < datasetParents; i++ {
		parent := fnv(index^i, intMix[i%16]) % rows
		fnvHash(intMix, cache[parent*hashWords:])
	}
	// Flatten the uint32 mix into a binary one and return
	for i, val := range intMix {
		binary.LittleEndian.PutUint32(mix[i*4:], val)
	}
	hasher(mix, mix)
	return mix
}

// generateDataset generates the entire ethash dataset for mining.
// This method places the result into dest in machine byte order.
func generateDataset(dest []uint32, epoch uint64, cache []uint32) {
	// Print some debug logs to allow analysis on low end devices
	logger := log.New("epoch", epoch)

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)

		logFn := logger.Debug
		if elapsed > 3*time.Second {
			logFn = logger.Info
		}
		logFn("Generated ethash verification cache", "elapsed", common.PrettyDuration(elapsed))
	}()

	// Figure out whether the bytes need to be swapped for the machine
	swapped := !isLittleEndian()

	// Convert our destination slice to a byte buffer
	var dataset []byte
	datasetHdr := (*reflect.SliceHeader)(unsafe.Pointer(&dataset))
	destHdr := (*reflect.SliceHeader)(unsafe.Pointer(&dest))
	datasetHdr.Data = destHdr.Data
	datasetHdr.Len = destHdr.Len * 4
	datasetHdr.Cap = destHdr.Cap * 4

	// Generate the dataset on many goroutines since it takes a while
	threads := runtime.NumCPU()
	size := uint64(len(dataset))

	var pend sync.WaitGroup
	pend.Add(threads)

	var progress atomic.Uint64
	for i := 0; i < threads; i++ {
		go func(id int) {
			defer pend.Done()

			// Create a hasher to reuse between invocations
			//hasherCB := makeHasher(sha3.NewLegacyKeccak512())
			hasherCB := makeHasher(blake3.New(64, nil))

			// Calculate the data segment this thread should generate
			batch := (size + hashBytes*uint64(threads) - 1) / (hashBytes * uint64(threads))
			first := uint64(id) * batch
			limit := first + batch
			if limit > size/hashBytes {
				limit = size / hashBytes
			}
			// Calculate the dataset segment
			percent := size / hashBytes / 100
			for index := first; index < limit; index++ {
				item := generateDatasetItem(cache, uint32(index), hasherCB)
				if swapped {
					swap(item)
				}
				copy(dataset[index*hashBytes:], item)

				if status := progress.Add(1); status%percent == 0 {
					logger.Info("Generating DAG in progress", "percentage", (status*100)/(size/hashBytes), "elapsed", common.PrettyDuration(time.Since(start)))
				}
			}
		}(i)
	}
	// Wait for all the generators to finish and return
	pend.Wait()
}

// hashimoto aggregates data from the full dataset in order to produce our final
// value for a particular header hash and nonce.
func hashimoto(hash []byte, nonce uint64, size uint64, lookup func(index uint32) []uint32) ([]byte, []byte) {
	// Calculate the number of theoretical rows (we use one buffer nonetheless)
	rows := uint32(size / mixBytes)

	// Combine header+nonce into a 40 byte seed
	seed := make([]byte, 40)
	copy(seed, hash)
	binary.LittleEndian.PutUint64(seed[32:], nonce)

	//seed = crypto.Keccak512(seed)
	b64 := blake3.Sum512(seed)
	seed = b64[:]
	seedHead := binary.LittleEndian.Uint32(seed)

	// Start the mix with replicated seed
	mix := make([]uint32, mixBytes/4)
	for i := 0; i < len(mix); i++ {
		mix[i] = binary.LittleEndian.Uint32(seed[i%16*4:])
	}
	// Mix in random dataset nodes
	temp := make([]uint32, len(mix))

	for i := 0; i < loopAccesses; i++ {
		parent := fnv(uint32(i)^seedHead, mix[i%len(mix)]) % rows
		for j := uint32(0); j < mixBytes/hashBytes; j++ {
			copy(temp[j*hashWords:], lookup(2*parent+j))
		}
		fnvHash(mix, temp)
	}
	// Compress mix
	for i := 0; i < len(mix); i += 4 {
		mix[i/4] = fnv(fnv(fnv(mix[i], mix[i+1]), mix[i+2]), mix[i+3])
	}
	mix = mix[:len(mix)/4]

	digest := make([]byte, common.HashLength)
	for i, val := range mix {
		binary.LittleEndian.PutUint32(digest[i*4:], val)
	}
	b32 := blake3.Sum256(append(seed, digest...))
	return digest, b32[:]
}

// hashimotoLight aggregates data from the full dataset (using only a small
// in-memory cache) in order to produce our final value for a particular header
// hash and nonce.
func hashimotoLight(size uint64, cache []uint32, hash []byte, nonce uint64) ([]byte, []byte) {
	hasherCB := makeHasher(sha3.NewLegacyKeccak512())

	lookup := func(index uint32) []uint32 {
		rawData := generateDatasetItem(cache, index, hasherCB)

		data := make([]uint32, len(rawData)/4)
		for i := 0; i < len(data); i++ {
			data[i] = binary.LittleEndian.Uint32(rawData[i*4:])
		}
		return data
	}
	return hashimoto(hash, nonce, size, lookup)
}

// hashimotoFull aggregates data from the full dataset (using the full in-memory
// dataset) in order to produce our final value for a particular header hash and
// nonce.
func hashimotoFull(dataset []uint32, hash []byte, nonce uint64) ([]byte, []byte) {
	lookup := func(index uint32) []uint32 {
		offset := index * hashWords
		return dataset[offset : offset+hashWords]
	}
	return hashimoto(hash, nonce, uint64(len(dataset))*4, lookup)
}

const maxEpoch = 2048
