// Copyright 2019 The go-ethereum Authors
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

package forkid

import (
	"bytes"
	"math"
	"testing"

	"github.com/rethereum-blockchain/go-rethereum/common"
	"github.com/rethereum-blockchain/go-rethereum/params"
	"github.com/rethereum-blockchain/go-rethereum/rlp"
)

// TestCreation tests that different genesis and fork rule combinations result in
// the correct fork ID.
func TestCreation(t *testing.T) {
	type testcase struct {
		head uint64
		time uint64
		want ID
	}
	tests := []struct {
		config  *params.ChainConfig
		genesis common.Hash
		cases   []testcase
	}{
		// Mainnet test cases
		{
			params.MainnetChainConfig,
			params.MainnetGenesisHash,
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0x61aefa70), Next: 1001}},                // Unsynced
				{1000, 0, ID{Hash: checksumToBytes(0x61aefa70), Next: 1001}},             // Last Frontier block
				{1001, 0, ID{Hash: checksumToBytes(0x7cc30c12), Next: 5503}},             // First Homestead block
				{1002, 0, ID{Hash: checksumToBytes(0x7cc30c12), Next: 5503}},             // Last Homestead block
				{5502, 0, ID{Hash: checksumToBytes(0x7cc30c12), Next: 5503}},             // First DAO block
				{5503, 0, ID{Hash: checksumToBytes(0xc04d6826), Next: 5507}},             // Last DAO block
				{5506, 0, ID{Hash: checksumToBytes(0xc04d6826), Next: 5507}},             // First Tangerine block
				{5507, 0, ID{Hash: checksumToBytes(0xfbb573dd), Next: 5519}},             // Last Tangerine block
				{5510, 0, ID{Hash: checksumToBytes(0xfbb573dd), Next: 5519}},             // First Spurious block
				{5519, 0, ID{Hash: checksumToBytes(0x1aebed3d), Next: 5521}},             // Last Spurious block
				{5521, 0, ID{Hash: checksumToBytes(0xfff37fb6), Next: 5527}},             // First Byzantium block
				{5526, 0, ID{Hash: checksumToBytes(0xfff37fb6), Next: 5527}},             // Last Byzantium block
				{5527, 0, ID{Hash: checksumToBytes(0x6f27ec43), Next: 13_524_557}},       // First and last Constantinople, first Petersburg block
				{1_000_000, 0, ID{Hash: checksumToBytes(0x6f27ec43), Next: 13_524_557}},  // Last Petersburg block
				{5_000_000, 0, ID{Hash: checksumToBytes(0x6f27ec43), Next: 13_524_557}},  // Last Petersburg block
				{10_000_000, 0, ID{Hash: checksumToBytes(0x6f27ec43), Next: 13_524_557}}, // Last Petersburg block
				{13_524_557, 0, ID{Hash: checksumToBytes(0xb09ff5ab), Next: 27_200_177}}, // First Istanbul and first Muir Glacier block
				{27_200_177, 0, ID{Hash: checksumToBytes(0x50ce6e77), Next: 40_725_107}}, // Last Istanbul and first Muir Glacier block
			},
		},
	}
	for i, tt := range tests {
		for j, ttt := range tt.cases {
			if have := NewID(tt.config, tt.genesis, ttt.head, ttt.time); have != ttt.want {
				t.Errorf("test %d, case %d: fork ID mismatch: have %x, want %x", i, j, have, ttt.want)
			}
		}
	}
}

// TestValidation tests that a local peer correctly validates and accepts a remote
// fork ID.
func TestValidation(t *testing.T) {
	// Config that has not timestamp enabled
	legacyConfig := *params.MainnetChainConfig
	legacyConfig.ShanghaiTime = nil

	tests := []struct {
		config *params.ChainConfig
		head   uint64
		time   uint64
		id     ID
		err    error
	}{
		//------------------
		// Block based tests
		//------------------

		// Local is mainnet Gray Glacier, remote announces the same. No future fork is announced.
		{&legacyConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},

		// Local is mainnet Gray Glacier, remote announces the same. Remote also announces a next fork
		// at block 0xffffffff, but that is uncertain.
		{&legacyConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: math.MaxUint64}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, but it's not yet aware of Petersburg (e.g. non updated node before the fork).
		// In this case we don't know if Petersburg passed yet or not.
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of Petersburg (e.g. updated node before the fork). We
		// don't know if Petersburg passed yet (will pass) or not.
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of some random fork (e.g. misconfigured Petersburg). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: math.MaxUint64}, nil},

		// Local is mainnet exactly on Petersburg, remote announces Byzantium + knowledge about Petersburg. Remote
		// is simply out of sync, accept.
		{&legacyConfig, 7280000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet Petersburg, remote announces Byzantium + knowledge about Petersburg. Remote
		// is simply out of sync, accept.
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet Petersburg, remote announces Spurious + knowledge about Byzantium. Remote
		// is definitely out of sync. It may or may not need the Petersburg update, we don't know yet.
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}, nil},

		// Local is mainnet Byzantium, remote announces Petersburg. Local is out of sync, accept.
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 0}, nil},

		// Local is mainnet Spurious, remote announces Byzantium, but is not aware of Petersburg. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		{&legacyConfig, 4369999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},

		// Local is mainnet Petersburg. remote announces Byzantium but is not aware of further forks.
		// Remote needs software update.
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, ErrRemoteStale},

		// Local is mainnet Petersburg, and isn't aware of more forks. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium, and is aware of Petersburg. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Petersburg, remote is Rinkeby Petersburg.
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0xafec6b27), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier, far in the future. Remote announces Gopherium (non existing fork)
		// at some future block 88888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		//
		// TODO(karalabe): This testcase will fail once mainnet gets timestamped forks, make legacy chain config
		{&legacyConfig, 88888888, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 88888888}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium. Remote is also in Byzantium, but announces Gopherium (non existing
		// fork) at block 7279999, before Petersburg. Local is incompatible.
		//
		// TODO(karalabe): This testcase will fail once mainnet gets timestamped forks, make legacy chain config
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7279999}, ErrLocalIncompatibleOrStale},

		//------------------------------------
		// Block to timestamp transition tests
		//------------------------------------

		// Local is mainnet currently in Gray Glacier only (so it's aware of Shanghai), remote announces
		// also Gray Glacier, but it's not yet aware of Shanghai (e.g. non updated node before the fork).
		// In this case we don't know if Shanghai passed yet or not.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},

		// Local is mainnet currently in Gray Glacier only (so it's aware of Shanghai), remote announces
		// also Gray Glacier, and it's also aware of Shanghai (e.g. updated node before the fork). We
		// don't know if Shanghai passed yet (will pass) or not.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}, nil},

		// Local is mainnet currently in Gray Glacier only (so it's aware of Shanghai), remote announces
		// also Gray Glacier, and it's also aware of some random fork (e.g. misconfigured Shanghai). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: math.MaxUint64}, nil},

		// Local is mainnet exactly on Shanghai, remote announces Gray Glacier + knowledge about Shanghai. Remote
		// is simply out of sync, accept.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}, nil},

		// Local is mainnet Shanghai, remote announces Gray Glacier + knowledge about Shanghai. Remote
		// is simply out of sync, accept.
		{params.MainnetChainConfig, 20123456, 1681338456, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}, nil},

		// Local is mainnet Shanghai, remote announces Arrow Glacier + knowledge about Gray Glacier. Remote
		// is definitely out of sync. It may or may not need the Shanghai update, we don't know yet.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}, nil},

		// Local is mainnet Gray Glacier, remote announces Shanghai. Local is out of sync, accept.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}, nil},

		// Local is mainnet Arrow Glacier, remote announces Gray Glacier, but is not aware of Shanghai. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		{params.MainnetChainConfig, 13773000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},

		// Local is mainnet Shanghai. remote announces Gray Glacier but is not aware of further forks.
		// Remote needs software update.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, ErrRemoteStale},

		// Local is mainnet Gray Glacier, and isn't aware of more forks. Remote announces Gray Glacier +
		// 0xffffffff. Local needs software update, reject.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(checksumUpdate(0xf0afd0e3, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier, and is aware of Shanghai. Remote announces Shanghai +
		// 0xffffffff. Local needs software update, reject.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(checksumUpdate(0xdce96c2d, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier, far in the future. Remote announces Gopherium (non existing fork)
		// at some future timestamp 8888888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		{params.MainnetChainConfig, 888888888, 1660000000, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1660000000}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier. Remote is also in Gray Glacier, but announces Gopherium (non existing
		// fork) at block 7279999, before Shanghai. Local is incompatible.
		{params.MainnetChainConfig, 19999999, 1667999999, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1667999999}, ErrLocalIncompatibleOrStale},

		//----------------------
		// Timestamp based tests
		//----------------------

		// Local is mainnet Shanghai, remote announces the same. No future fork is announced.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}, nil},

		// Local is mainnet Shanghai, remote announces the same. Remote also announces a next fork
		// at time 0xffffffff, but that is uncertain.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xdce96c2d), Next: math.MaxUint64}, nil},

		// Local is mainnet currently in Shanghai only (so it's aware of Cancun), remote announces
		// also Shanghai, but it's not yet aware of Cancun (e.g. non updated node before the fork).
		// In this case we don't know if Cancun passed yet or not.
		//
		// TODO(karalabe): Enable this when Cancun is specced
		//{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}, nil},

		// Local is mainnet currently in Shanghai only (so it's aware of Cancun), remote announces
		// also Shanghai, and it's also aware of Cancun (e.g. updated node before the fork). We
		// don't know if Cancun passed yet (will pass) or not.
		//
		// TODO(karalabe): Enable this when Cancun is specced and update next timestamp
		//{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, nil},

		// Local is mainnet currently in Shanghai only (so it's aware of Cancun), remote announces
		// also Shanghai, and it's also aware of some random fork (e.g. misconfigured Cancun). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		//
		// TODO(karalabe): Enable this when Cancun is specced
		//{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: math.MaxUint64}, nil},

		// Local is mainnet exactly on Cancun, remote announces Shanghai + knowledge about Cancun. Remote
		// is simply out of sync, accept.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update local head and time, next timestamp
		// {params.MainnetChainConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, nil},

		// Local is mainnet Cancun, remote announces Shanghai + knowledge about Cancun. Remote
		// is simply out of sync, accept.
		// TODO(karalabe): Enable this when Cancun is specced, update local head and time, next timestamp
		//{params.MainnetChainConfig, 21123456, 1678123456, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, nil},

		// Local is mainnet Prague, remote announces Shanghai + knowledge about Cancun. Remote
		// is definitely out of sync. It may or may not need the Prague update, we don't know yet.
		//
		// TODO(karalabe): Enable this when Cancun **and** Prague is specced, update all the numbers
		//{params.MainnetChainConfig, 0, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}, nil},

		// Local is mainnet Shanghai, remote announces Cancun. Local is out of sync, accept.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update remote checksum
		//{params.MainnetChainConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x00000000), Next: 0}, nil},

		// Local is mainnet Shanghai, remote announces Cancun, but is not aware of Prague. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		//
		// TODO(karalabe): Enable this when Cancun **and** Prague is specced, update remote checksum
		//{params.MainnetChainConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x00000000), Next: 0}, nil},

		// Local is mainnet Cancun. remote announces Shanghai but is not aware of further forks.
		// Remote needs software update.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update local head and time
		//{params.MainnetChainConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}, ErrRemoteStale},

		// Local is mainnet Shanghai, and isn't aware of more forks. Remote announces Shanghai +
		// 0xffffffff. Local needs software update, reject.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(checksumUpdate(0xdce96c2d, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, and is aware of Cancun. Remote announces Cancun +
		// 0xffffffff. Local needs software update, reject.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update remote checksum
		//{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(checksumUpdate(0x00000000, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, remote is random Shanghai.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0x12345678), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, far in the future. Remote announces Gopherium (non existing fork)
		// at some future timestamp 8888888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		{params.MainnetChainConfig, 88888888, 8888888888, ID{Hash: checksumToBytes(0xdce96c2d), Next: 8888888888}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai. Remote is also in Shanghai, but announces Gopherium (non existing
		// fork) at timestamp 1668000000, before Cancun. Local is incompatible.
		//
		// TODO(karalabe): Enable this when Cancun is specced
		//{params.MainnetChainConfig, 20999999, 1677999999, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, ErrLocalIncompatibleOrStale},
	}
	for i, tt := range tests {
		filter := newFilter(tt.config, params.MainnetGenesisHash, func() (uint64, uint64) { return tt.head, tt.time })
		if err := filter(tt.id); err != tt.err {
			t.Errorf("test %d: validation error mismatch: have %v, want %v", i, err, tt.err)
		}
	}
}

// Tests that IDs are properly RLP encoded (specifically important because we
// use uint32 to store the hash, but we need to encode it as [4]byte).
func TestEncoding(t *testing.T) {
	tests := []struct {
		id   ID
		want []byte
	}{
		{ID{Hash: checksumToBytes(0), Next: 0}, common.Hex2Bytes("c6840000000080")},
		{ID{Hash: checksumToBytes(0xdeadbeef), Next: 0xBADDCAFE}, common.Hex2Bytes("ca84deadbeef84baddcafe,")},
		{ID{Hash: checksumToBytes(math.MaxUint32), Next: math.MaxUint64}, common.Hex2Bytes("ce84ffffffff88ffffffffffffffff")},
	}
	for i, tt := range tests {
		have, err := rlp.EncodeToBytes(tt.id)
		if err != nil {
			t.Errorf("test %d: failed to encode forkid: %v", i, err)
			continue
		}
		if !bytes.Equal(have, tt.want) {
			t.Errorf("test %d: RLP mismatch: have %x, want %x", i, have, tt.want)
		}
	}
}
