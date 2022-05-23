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

package snapshot

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)
// *** modify to support nft transaction 20211217 begin ***

// Account is a modified version of a state.Account, where the root is replaced
// with a byte slice. This format can be used to represent full-consensus format
// or slim-snapshot format which replaces the empty root and code hash as nil
// byte slice.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     []byte
	CodeHash []byte

	PledgedBalance *big.Int
	//Owner common.Address
	// whether the account has a NFT exchanger
	ExchangerFlag bool
	BlockNumber *big.Int
	ExchangerBalance *big.Int
	VoteWeight *big.Int
	// The ratio that exchanger get.
	FeeRate uint32
	ExchangerName string
	ExchangerURL string
	// ApproveAddress have the right to handle all nfts of the account
	ApproveAddressList []common.Address
	// NFTBalance is the nft number that the account have
	NFTBalance uint64
	AccountNFT
}
type AccountNFT struct {
	//Account
	Name string
	Symbol string
	Price *big.Int
	Direction uint8 // 0:未交易,1:买入,2:卖出
	Owner common.Address
	NFTApproveAddressList common.Address
	//Auctions map[string][]common.Address
	// MergeLevel is the level of NFT merged
	MergeLevel uint8

	Creator common.Address
	Royalty uint32
	Exchanger common.Address
	MetaURL string
}

// SlimAccount converts a state.Account content into a slim snapshot account
func SlimAccount(nonce uint64,
	balance *big.Int,
	root common.Hash,
	codehash []byte,
	pledgedbalance *big.Int,
	exchangerflag bool,
	blocknumber *big.Int,
	exchangerbalance *big.Int,
	voteweight *big.Int,
	feerate uint32,
	exchangername string,
	exchangerurl string,
	approveaddresslist []common.Address,
	nftbalance uint64,
	name string,
	symbol string,
	price *big.Int,
	direction uint8,
	owner common.Address,
	nftapproveaddresslist common.Address,
	mergelevel uint8,
	creator common.Address,
	royalty uint32,
	exchanger common.Address,
	metaurl string) Account {
//func SlimAccount(nonce uint64, balance *big.Int, root common.Hash, codehash []byte) Account {
	slim := Account{
		Nonce:   nonce,
		Balance: balance,
		PledgedBalance: pledgedbalance,
		ExchangerFlag: exchangerflag,
		BlockNumber: blocknumber,
		ExchangerBalance: exchangerbalance,
		VoteWeight: voteweight,
		FeeRate: feerate,
		ExchangerName: exchangername,
		ExchangerURL: exchangerurl,
		NFTBalance: nftbalance,
		AccountNFT: AccountNFT{
			Name: name,
			Symbol: symbol,
			Price: price,
			Direction: direction,
			Owner: owner,
			MergeLevel: mergelevel,
			Creator: creator,
			Royalty: royalty,
			Exchanger: exchanger,
			MetaURL: metaurl,
		},
	}
	slim.ApproveAddressList = append(slim.ApproveAddressList, approveaddresslist...)
	//slim.NFTApproveAddressList = append(slim.NFTApproveAddressList, nftapproveaddresslist...)
	slim.NFTApproveAddressList = nftapproveaddresslist
// *** modify to support nft transaction 20211217 end ***
	if root != emptyRoot {
		slim.Root = root[:]
	}
	if !bytes.Equal(codehash, emptyCode[:]) {
		slim.CodeHash = codehash
	}
	return slim
}

// *** modify to support nft transaction 20211217 begin ***

// SlimAccountRLP converts a state.Account content into a slim snapshot
// version RLP encoded.
func SlimAccountRLP(nonce uint64,
	balance *big.Int,
	root common.Hash,
	codehash []byte,
	pledgedbalance *big.Int,
	exchangerflag bool,
	blocknumber *big.Int,
	exchangerbalance *big.Int,
	voteweight *big.Int,
	feerate uint32,
	exchangername string,
	exchangerurl string,
	approveaddresslist []common.Address,
	nftbalance uint64,
	name string,
	symbol string,
	price *big.Int,
	direction uint8,
	owner common.Address,
	nftapproveaddresslist common.Address,
	mergelevel uint8,
	creator common.Address,
	royalty uint32,
	exchanger common.Address,
	metaurl string) []byte {
	data, err := rlp.EncodeToBytes(SlimAccount(nonce,
		balance,
		root,
		codehash,
		pledgedbalance,
		exchangerflag,
		blocknumber,
		exchangerbalance,
		voteweight,
		feerate,
		exchangername,
		exchangerurl,
		approveaddresslist,
		nftbalance,
		name,
		symbol,
		price,
		direction,
		owner,
		nftapproveaddresslist,
		mergelevel,
		creator,
		royalty,
		exchanger,
		metaurl))
//func SlimAccountRLP(nonce uint64, balance *big.Int, root common.Hash, codehash []byte) []byte {
//	data, err := rlp.EncodeToBytes(SlimAccount(nonce, balance, root, codehash))
// *** modify to support nft transaction 20211217 end ***
	if err != nil {
		panic(err)
	}
	return data
}

// FullAccount decodes the data on the 'slim RLP' format and return
// the consensus format account.
func FullAccount(data []byte) (Account, error) {
	var account Account
	if err := rlp.DecodeBytes(data, &account); err != nil {
		return Account{}, err
	}
	if len(account.Root) == 0 {
		account.Root = emptyRoot[:]
	}
	if len(account.CodeHash) == 0 {
		account.CodeHash = emptyCode[:]
	}
	return account, nil
}

// FullAccountRLP converts data on the 'slim RLP' format into the full RLP-format.
func FullAccountRLP(data []byte) ([]byte, error) {
	account, err := FullAccount(data)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(account)
}
