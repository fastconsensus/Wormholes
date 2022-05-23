package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

type MintDeep struct {
	UserMint *big.Int
	OfficialMint *big.Int
	//ExchangeList SNFTExchangeList
}

type SNFTExchange struct {
	InjectedInfo
	NFTAddress common.Address
	MergeLevel uint8
	CurrentMintAddress common.Address
	BlockNumber *big.Int

}
type InjectedInfo struct {
	MetalUrl string
	Royalty uint32
	Creator string
}

type SNFTExchangeList struct {
	SNFTExchanges []*SNFTExchange
}
func (ex *SNFTExchange) MinNFTAddress() common.Address {
	return ex.NFTAddress
}
func (ex *SNFTExchange) MaxNFTAddress() common.Address {
	if ex.MergeLevel == 0 {
		return ex.NFTAddress
	}
	minAddrInt := big.NewInt(0)
	minAddrInt.SetBytes(ex.NFTAddress.Bytes())
	nftNumber := math.BigPow(256, int64(ex.MergeLevel))
	maxAddrInt := big.NewInt(0)
	maxAddrInt.Add(minAddrInt, nftNumber)
	maxAddr := common.BytesToAddress(maxAddrInt.Bytes())
	return maxAddr
}
func (list *SNFTExchangeList) PopAddress(blocknumber *big.Int) (common.Address, *InjectedInfo, bool) {
	if len(list.SNFTExchanges) >0 {
		log.Info("PopAddress()", "SNFTExchanges[0].BlockNumber=", list.SNFTExchanges[0].BlockNumber.Uint64())
		log.Info("PopAddress()", "-----------------blocknumber=", blocknumber.Uint64())
		if list.SNFTExchanges[0].BlockNumber.Cmp(blocknumber) >= 0 {
			return common.Address{}, nil, false
		}
		addr := list.SNFTExchanges[0].CurrentMintAddress
		InjectedInfo := &InjectedInfo{
			MetalUrl: list.SNFTExchanges[0].MetalUrl,
			Royalty:  list.SNFTExchanges[0].Royalty,
			Creator: list.SNFTExchanges[0].Creator,
		}
		if list.SNFTExchanges[0].CurrentMintAddress == list.SNFTExchanges[0].MaxNFTAddress() {
			if len(list.SNFTExchanges) > 1 {
				list.SNFTExchanges = list.SNFTExchanges[1:]
			} else {
				list.SNFTExchanges = list.SNFTExchanges[:0]
			}
		} else {
			currentMintInt := new(big.Int).SetBytes(list.SNFTExchanges[0].CurrentMintAddress.Bytes())
			currentMintInt.Add(currentMintInt, big.NewInt(1))
			list.SNFTExchanges[0].CurrentMintAddress = common.BytesToAddress(currentMintInt.Bytes())
		}
		return addr, InjectedInfo, true
	}
	return common.Address{}, nil, false
}

type PledgedToken struct {
	Address common.Address
	Amount *big.Int
	Flag bool
}

type InjectedOfficialNFT struct {
	Dir string 				`json:"dir"`
	StartIndex *big.Int		`json:"start_index"`
	Number uint64			`json:"number"`
	Royalty uint32			`json:"royalty"`
	Creator string 			`json:"creator"`
}

type InjectedOfficialNFTList struct {
	InjectedOfficialNFTs []*InjectedOfficialNFT
}

func (list *InjectedOfficialNFTList) GetInjectedInfo(addr common.Address) *InjectedOfficialNFT {
	maskB, _ := big.NewInt(0).SetString("8000000000000000000000000000000000000000", 16)
	addrInt := new(big.Int).SetBytes(addr.Bytes())
	addrInt.Sub(addrInt, maskB)
	tempInt := new(big.Int)
	for _, injectOfficialNFT := range list.InjectedOfficialNFTs {
		if injectOfficialNFT.StartIndex.Cmp(addrInt) == 0 {
			return injectOfficialNFT
		}
		if injectOfficialNFT.StartIndex.Cmp(addrInt) < 0 {
			tempInt.SetInt64(0)
			tempInt.Add(injectOfficialNFT.StartIndex, new(big.Int).SetUint64(injectOfficialNFT.Number))
			if tempInt.Cmp(addrInt) >= 0 {
				return injectOfficialNFT
			}
		}
	}

	return nil
}

func (list *InjectedOfficialNFTList) DeleteExpireElem(num *big.Int) {
	var index int
	maskB, _ := big.NewInt(0).SetString("8000000000000000000000000000000000000000", 16)
	for k, injectOfficialNFT := range list.InjectedOfficialNFTs {
		sum := new(big.Int).Add(injectOfficialNFT.StartIndex, new(big.Int).SetUint64(injectOfficialNFT.Number))
		sum.Add(sum, maskB)
		if sum.Cmp(num) > 0 {
			index = k
			break
		}
	}

	list.InjectedOfficialNFTs = list.InjectedOfficialNFTs[index:]
}

func (list *InjectedOfficialNFTList) RemainderNum(addrInt *big.Int) uint64 {
	var sum uint64
	maskB, _ := big.NewInt(0).SetString("8000000000000000000000000000000000000000", 16)
	tempInt := new(big.Int)
	for _, injectOfficialNFT := range list.InjectedOfficialNFTs {
		if injectOfficialNFT.StartIndex.Cmp(addrInt) >= 0 {
			sum = sum + injectOfficialNFT.Number
		}
		if injectOfficialNFT.StartIndex.Cmp(addrInt) < 0 {
			tempInt.SetInt64(0)
			tempInt.Add(injectOfficialNFT.StartIndex, new(big.Int).SetUint64(injectOfficialNFT.Number))
			tempInt.Add(tempInt, maskB)
			if tempInt.Cmp(addrInt) >= 0 {
				sum = sum  + new(big.Int).Sub(tempInt, addrInt).Uint64()
			}
		}
	}

	return sum
}

func (list *InjectedOfficialNFTList) MaxIndex() *big.Int {
	max := big.NewInt(0)
	for _, injectOfficialNFT := range list.InjectedOfficialNFTs {
		index := new(big.Int).Add(injectOfficialNFT.StartIndex, new(big.Int).SetUint64(injectOfficialNFT.Number))
		if index.Cmp(max) > 0 {
			max.Set(index)
		}
	}

	return max
}

// Wormholes struct for handling NFT transactions
type Wormholes struct {
	Type uint8					`json:"type"`
	NFTAddress string			`json:"nft_address"`
	Exchanger string			`json:"exchanger"`
	Royalty uint32				`json:"royalty"`
	MetaURL string				`json:"meta_url"`
	//ApproveAddress string		`json:"approve_address"`
	FeeRate uint32				`json:"fee_rate"`
	Name string					`json:"name"`
	Url string					`json:"url"`
	Dir string					`json:"dir"`
	StartIndex	string			`json:"start_index"`
	Number uint64				`json:"number"`
	Buyer Payload				`json:"buyer"`
	Seller1 Payload				`json:"seller1"`
	Seller2 MintSellPayload		`json:"seller2"`
	ExchangerAuth ExchangerPayload				`json:"exchanger_auth"`
	Creator	string				`json:"creator"`
	Version string				`json:"version"`
}

type Payload struct {
	Amount string				`json:"price"`
	NFTAddress string			`json:"nft_address"`
	Exchanger string			`json:"exchanger"`
	BlockNumber string			`json:"block_number"`
	Seller string				`json:"seller"`
	Sig string					`json:"sig"`
}

type MintSellPayload struct {
	Amount string				`json:"price"`
	Royalty string				`json:"royalty"`
	MetaURL string				`json:"meta_url"`
	ExclusiveFlag string 		`json:"exclusive_flag"`
	Exchanger string			`json:"exchanger"`
	BlockNumber string			`json:"block_number"`
	Sig string					`json:"sig"`
}

type ExchangerPayload struct {
	ExchangerOwner string		`json:"exchanger_owner"`
	To string					`json:"to"`
	BlockNumber string			`json:"block_number"`
	Sig	string					`json:"sig"`
}
// *** modify to support nft transaction 20211215 end ***


type NominatedOfficialNFT struct {
	InjectedOfficialNFT
	Address common.Address
}