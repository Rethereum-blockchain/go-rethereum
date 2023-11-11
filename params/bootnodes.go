// Copyright 2015 The go-ethereum Authors
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

package params

import "github.com/rethereum-blockchain/go-rethereum/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Rethereum Bootnodes
	"enode://b3d8c6ad187f54860bd288e8e343c5cb05db023b3a74a4cd9d85cae3e2677074f92b3afecfd2bb445f9cba151848d3294abff9bedcee5d437ff161300f5144e9@77.100.75.201:30303", // Dev
	"enode://301c2d2d622fe3d49f9a9d5a294a1a65ce0f686a10b5b6ea2e965533b7e84ecea25f1f2eec78e6fa948ca129ec5f9a8fe731d9641df0163e4847ded09dbfd1e4@54.36.108.60:30303",  // Explorer
	// Communtiy Bootnodes
	"enode://959f6378ee6162f57977e5e6ab8dd56bd8ef5d1bc2a1bb01c6b41cfc2d07ea490d4c939c7625f13886c684b221a9c3e710e4a66a718a3231c40d2536c344df9d@27.254.39.27:30308",
	"enode://e82bf286f09a7b86f5528a0e7c29928a8bb0bf9416d9678a91da9e2729480700a71777490ed115cad82b9f75268fc1f9a0d9483bb65acd6665708778c2d035f5@178.234.84.24:30303?discport=1337",
	"enode://fe072785d5044f22b393df8a364dcc92d927b9f88aff14bff2484db20caa8350a07df3b9b1f0fb0b222304f426ab887ad9829bff6948aba84e3b5f1776b8dd52@195.201.122.219:30303",
}

// KrontosBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Krontos test network.
var KrontosBootnodes = []string{
	// EF DevOps
	"enode://9a392272f3688e8fa414c88bd9a341690acc651078d4e22551f1161b3a96e28f60a09fd39bae0a78f63388aa62f549011853444f195cc6b0db85954745b017fa@77.100.75.201:30305", // krontos-bootnode-1-nyc3
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{}

// GoerliBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Görli test network.
var GoerliBootnodes = []string{}

var V5Bootnodes = []string{
	// Teku team's bootnode
	"enr:-KW4QPeYjGcWLIq9Jtmly0R8wlhPJWz5D6lWwUIQ_3ycVLUED3puuQhqMtQ-osu_IYgdIkoMcR7Qgd077Fbe9-rOj1iGAYlu4Ek8g2V0aMnIhEIH29-CFUmCaWSCdjSCaXCETWRLyYlzZWNwMjU2azGhAkEGf2xmqbqacLe5pVhVCImq3VwB8CfxKHxXCSzCMNzthHNuYXDAg3RjcIJ2X4N1ZHCCdl8",
	"enr:-KG4QDyytgmE4f7AnvW-ZaUOIi9i79qX4JwjRAiXBZCU65wOfBu-3Nb5I7b_Rmg3KCOcZM_C3y5pg7EBU5XGrcLTduQEhGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQ2_DUbiXNlY3AyNTZrMaEDKnz_-ps3UUOfHWVYaskI5kWYO_vtYMGYCQRAR3gHDouDdGNwgiMog3VkcIIjKA",
}

const dnsPrefix = "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@"

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	var net string
	switch genesis {
	case MainnetGenesisHash:
		net = "mainnet"
	case RinkebyGenesisHash:
		net = "rinkeby"
	case GoerliGenesisHash:
		net = "goerli"
	case KrontosGenesisHash:
		net = "krontos"
	default:
		return ""
	}
	return dnsPrefix + protocol + "." + net + ".ethdisco.net"
}
