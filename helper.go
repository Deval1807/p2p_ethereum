package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type NetworkInfo struct {
	enodes      []*enode.Node
	rpcUrl      string
	genesisHash common.Hash
	forkId      forkid.ID
}

// getEnodes parses string bootnodes into enode struct
func getEnodes(bootnodes []string) []*enode.Node {
	enodes := []*enode.Node{}
	for _, bootnode := range bootnodes {
		node, err := enode.ParseV4(bootnode)
		if err != nil {
			return nil
		}
		enodes = append(enodes, node)
	}

	return enodes
}

// getNetworkInfo returns the network information for eth and polygon networks
func getNetworkInfo(network string) *NetworkInfo {
	switch network {
	case "eth_mainnet":
		// Add more bootnodes from https://github.com/maticnetwork/bor/blob/develop/params/bootnodes.go#L23
		bootnodes := []string{"enode://d860a01f9722d78051619d1e2351aba3f43f943f6f00718d1b9baa4101932a1f5011f16bb2b1bb35db20d6fe28fa0bf09636d26a87d31de9ec6203eeedb1f666@18.138.108.67:30303"}
		return &NetworkInfo{
			enodes:      getEnodes(bootnodes),
			rpcUrl:      "https://eth.public-rpc.com",                                                           // public rpc
			genesisHash: common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"), // Hash of genesis (0th) block
			forkId:      forkid.ID{Hash: [4]byte{159, 61, 34, 84}},
		}
	case "polygon_mainnet":
		// Add more bootnodes from https://github.com/maticnetwork/bor/blob/develop/params/bootnodes.go#L87
		bootnodes := []string{"enode://b8f1cc9c5d4403703fbf377116469667d2b1823c0daf16b7250aa576bacf399e42c3930ccfcb02c5df6879565a2b8931335565f0e8d3f8e72385ecf4a4bf160a@3.36.224.80:30303"}
		return &NetworkInfo{
			enodes:      getEnodes(bootnodes),
			rpcUrl:      "https://polygon-rpc.com",                                                              // public rpc
			genesisHash: common.HexToHash("0xa9c28ce2141b56c474f1dc504bee9b01eb1bd7d1a507580d5519d4437a97de1b"), // Hash of genesis (0th) block
			forkId:      forkid.ID{Hash: [4]byte{240, 151, 188, 19}},
		}
	}

	return nil
}

// getLatestBlockAndChainId will get the latest block and chain id from an RPC provider.
func getLatestBlockAndChainId(url string) (*types.Block, *big.Int, error) {
	eth, err := ethclient.Dial(url)
	if err != nil {
		return nil, nil, err
	}
	defer eth.Close()
	chainId, err := eth.ChainID(context.Background())
	if err != nil {
		return nil, nil, err
	}
	block, err := eth.BlockByNumber(context.Background(), nil)
	return block, chainId, err
}

// conn represents an individual connection with a peer.
type conn struct {
	node *enode.Node
	rw   p2p.MsgReadWriter
}

// EthProtocolOptions is the options used when creating a new eth protocol.
type EthProtocolOptions struct {
	GenesisHash common.Hash
	NetworkID   uint64
	ForkID      forkid.ID
	Head        *types.Block
}
