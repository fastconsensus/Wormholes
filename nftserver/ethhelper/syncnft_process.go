package ethhelper

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	common2 "github.com/nftexchange/nftserver/ethhelper/common"
	"github.com/nftexchange/nftserver/ethhelper/database"
	_ "image/gif"
	_ "image/jpeg"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	colChan   chan *database.Collection
	nftChan   chan *database.Nft
	nftTxChan chan *database.NftTx
)

type Nft struct {
	Uri  string `json:"uri"`
	Desc string `json:"description"`
	Name string `json:"name"`
	//Attributes []Attribute `json:"attributes"`
	Contract string `json:"contract"`
	TokenId  string `json:"tokenId"`
	Img      string `json:"image"`
}

type NftTxs []*database.NftTx

type NftTxsList []NftTxs

func (n NftTxsList) Len() int           { return len(n) }
func (n NftTxsList) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n NftTxsList) Less(i, j int) bool { return n[i][0].TransactionIndex < n[j][0].TransactionIndex }

func NftDbProcess() {
	colChan = make(chan *database.Collection, 200)
	nftChan = make(chan *database.Nft, 500)
	nftTxChan = make(chan *database.NftTx, 800)
	for {
		select {
		case col := <-colChan:
			col.Insert()
		case nft := <-nftChan:
			nft.Insert()
		case tx := <-nftTxChan:
			tx.Insert()
		default:
		}
	}
}

func (n *Nft) toModel() (ret database.Nft) {
	ret.Img = n.Img
	ret.Uri = n.Uri
	ret.Name = n.Name
	ret.Desc = n.Desc
	ret.TokenId = n.TokenId
	ret.Contract = n.Contract
	//if len(n.Attributes) > 0 {
	//	data, _ := json.Marshal(n.Attributes)
	//}
	return ret
}

type Connector struct {
	ctx  context.Context
	conn *ethclient.Client
	nft  *common2.Commoninterface
}

func NewConnector(addr string) *Connector {
	var (
		coinAddr = common.HexToAddress(addr)
	)
	conn, err := ethclient.Dial(common2.MainPoint)
	if err != nil {
		return nil
	}
	coin, err := common2.NewCommoninterface(coinAddr, conn)
	if err != nil {
		return nil
	}
	return &Connector{
		ctx:  context.Background(),
		conn: conn,
		nft:  coin,
	}
}

// SyncNftFromChain
// @description ??????nft????????????????????? ??????
// @auth chen.gang 2021/10/20 18:36
// @param num-????????? isFetch-?????????????????????   ?????????true, buyResultCh-???????????????????????????
// @return ret err
func SyncNftFromChain(num string, isFetch bool, buyResultCh chan<- []*database.NftTx, transferCh chan<- *WethTransfer, approveCh chan<- *WethTransfer, endCh chan<- bool) {
	var (
		tmp    big.Int
		col    database.Collection
		number string
	)

	tmp.SetString(num, 0)
	number = "0x" + strconv.FormatInt(tmp.Int64(), 16)
	contractType := ""
	if b, err := common2.GetBlock(number); err == nil {
		txMap := make(map[string]common2.Tx)
		//????????????????????????
		for _, tx := range b.Transactions {
			txMap[tx.Hash] = tx
			//NFT????????????
			if !isFetch && tx.To == "" {
				if receipt, er := common2.TransactionReceipt(tx.Hash); er == nil {
					re := contractCall(receipt.ContractAddress, erc721Input)
					tmp.SetString(re, 0)
					if tmp.Uint64() != 1 {
						re = contractCall(receipt.ContractAddress, erc1155Input)
						tmp.SetString(re, 0)
						if tmp.Uint64() == 1 {
							contractType = "ERC1155"
						}
					} else {
						contractType = "ERC721"
					}
					if contractType != "" {
						if contractType == "ERC721" {
							instance := NewConnector(receipt.ContractAddress)
							result, err := instance.nft.Name(&bind.CallOpts{Context: context.Background()})
							if err != nil {
							}
							col.Name = result
						}
						col.CreateTs = b.Ts
						col.CreateHash = tx.Hash
						col.UserAddr = tx.From
						col.ContractType = contractType
						col.ContractAddr = receipt.ContractAddress
						colChan <- &col
						//todo insert  nft collection
					}
				}
			}
		}
		type nftExpansion struct {
			value *big.Int
			nfts  NftTxs
		}
		nftMap := make(map[string]*nftExpansion, 0)
		txLogMap := make(map[string][]common2.Log, 0)

		//???????????????????????????tx??????????????????  ?????? ????????????1155 721??????
		//???????????????transfer log
		logs, err := common2.GetLogs(common2.LogFilter{FromBlock: number, ToBlock: number, Topics: []string{erc721TransferEvent}})
		logs1, err := common2.GetLogs(common2.LogFilter{FromBlock: number, ToBlock: number, Topics: []string{erc1155TransferEvent}})
		log2, err := common2.GetLogs(common2.LogFilter{FromBlock: number, ToBlock: number, Topics: []string{tokenApproveEvent}})
		//weth????????????  ????????????weth10   ??????????????????wet9
		for k := 0; k < len(log2); k++ {

			if strings.ToLower(log2[k].Address) != weth10 {
				continue
			}
			var transfer WethTransfer
			tmp.SetString(log2[k].Topics[2], 0)
			transfer.To = "0x" + hex.EncodeToString(tmp.Bytes())
			tmp.SetString(log2[k].Data, 0)
			transfer.Value = tmp.String()
			tmp.SetString(log2[k].Topics[1], 0)
			transfer.From = "0x" + hex.EncodeToString(tmp.Bytes())
			approveCh <- &transfer
		}
		logs = append(logs, logs1...)

		if err != nil {
			fmt.Println("GetLogs err:" + err.Error())
		}

		for _, logT := range logs {
			if _, b := txLogMap[logT.TxHash]; !b {
				txLogMap[logT.TxHash] = []common2.Log{}
			}
			txLogMap[logT.TxHash] = append(txLogMap[logT.TxHash], logT)
		}

		for _, logArr := range txLogMap {
			//???????????????????????????erc721????????????
			var buyer common.Address
			//????????????????????????????????????nft??????
			indexMp := make(map[int]int)
			for i, logT := range logArr {
				//weth??????????????????  ????????????weth10   ??????????????????wet9
				if strings.ToLower(logT.Address) == weth10 {
					var transfer WethTransfer
					tmp.SetString(logT.Data, 0)
					transfer.Value = tmp.String()
					tmp.SetString(logT.Topics[1], 0)
					transfer.From = "0x" + hex.EncodeToString(tmp.Bytes())
					tmp.SetString(logT.Topics[2], 0)
					transfer.To = "0x" + hex.EncodeToString(tmp.Bytes())
					transferCh <- &transfer
				}

				if len(logT.Topics) != 4 {
					continue
				}
				//???????????????????????????erc721?????????
				re := contractCall(logT.Address, erc721Input)
				tmp.SetString(re, 0)
				if tmp.Uint64() != 1 {
					re = contractCall(logT.Address, erc1155Input)
					tmp.SetString(re, 0)
					if tmp.Uint64() != 1 {
						continue
					}
					//1155
					indexMp[i] = 2
				} else {
					//721
					indexMp[i] = 1
				}
				if indexMp[i] == 1 {
					tmp.SetString(logT.Topics[2], 0)
				} else {
					tmp.SetString(logT.Topics[3], 0)
				}

				buyer = common.BytesToAddress(tmp.Bytes())
			}

			//???erc721??????1155????????????????????????  ?????????????????????
			if buyer == (common.Address{}) {
				continue
			}
			for i, logT := range logArr {
				currentTx := txMap[logT.TxHash]
				if i == 0 {
					var valueBig big.Int
					valueBig.SetString(currentTx.Value, 0)
					nftMap[logT.TxHash] = &nftExpansion{value: &valueBig}
				}
				//weth  balance  cost
				if strings.ToLower(logT.Address) == weth9 || strings.ToLower(logT.Address) == weth10 {
					tmp.SetString(logT.Topics[1], 0)
					if buyer != common.BytesToAddress(tmp.Bytes()) {
						continue
					}
					tmp.SetString(logT.Data, 0)
					nftMap[logT.TxHash].value.Add(nftMap[logT.TxHash].value, &tmp)
					continue
				}
				//??????nft Transfer??????
				if len(logT.Topics) != 4 || indexMp[i] == 0 {
					continue
				}

				if _, b := nftMap[logT.TxHash]; !b {
					nftMap[logT.TxHash] = &nftExpansion{}
				}
				var obj database.NftTx

				if indexMp[i] == 1 {
					tmp.SetString(logT.Topics[3], 0)
					obj.TokenId = tmp.String()
				} else if indexMp[i] == 2 {
					tmp.SetString(logT.Data[:66], 0)
					obj.TokenId = tmp.String()
				}
				if indexMp[i] == 1 {
					tmp.SetString(logT.Topics[1], 0)
				} else if indexMp[i] == 2 {
					tmp.SetString(logT.Topics[2], 0)
				}

				if !isFetch && tmp.Uint64() == 0 {
					tmp.SetString(obj.TokenId, 0)
					if !uploadNft(logT.Address, &tmp) {
						continue
					}
				} else {
					if tmp.Uint64() != 0 {
						obj.From = common.BytesToAddress(tmp.Bytes()).String()
					}
				}

				if indexMp[i] == 1 {
					tmp.SetString(logT.Topics[2], 0)
				} else if indexMp[i] == 2 {
					tmp.SetString(logT.Topics[3], 0)
				}
				obj.To = common.BytesToAddress(tmp.Bytes()).String()
				obj.Contract = logT.Address
				tmp.SetString(b.Ts, 0)
				obj.Ts = tmp.String()
				obj.TxHash = logT.TxHash
				tmp.SetString(logT.BlockNumber, 0)
				obj.BlockNumber = tmp.String()
				tmp.SetString(logT.TransactionIndex, 0)
				obj.TransactionIndex = tmp.String()
				nftMap[logT.TxHash].nfts = append(nftMap[logT.TxHash].nfts, &obj)
			}
		}
		//map??????  ?????????????????????tx_index??????
		var sortedTxData NftTxsList
		for _, obj := range nftMap {
			if len(obj.nfts) > 0 {
				sortedTxData = append(sortedTxData, obj.nfts)
			}
		}
		sort.Sort(sortedTxData)

		for _, obj := range sortedTxData {
			if len(obj) == 0 {
				continue
			}
			value := nftMap[obj[0].TxHash].value
			//??????????????? ??????????????????
			//from ==""
			var nftCount int64
			var price big.Int
			tokenMap := make(map[string]bool)
			for _, tx := range obj {
				if _, bb := tokenMap[tx.TokenId]; tx.From != "" && !bb {
					nftCount++
					tokenMap[tx.TokenId] = true
				}
			}
			if nftCount > 1 {
				price.Div(value, new(big.Int).SetInt64(nftCount))
			} else {
				price = *value
			}

			for _, tx := range obj {
				tx.Value = fmt.Sprintf("%v", price.Uint64())
				//todo insert nft transfer log
				if !isFetch {
					nftTxChan <- tx
				}
			}
			if isFetch {
				buyResultCh <- obj
				continue
			}
		}
	} else {
		log.Println("GetBlock err:", err.Error())
	}
	endCh <- true
}
func uploadNft(contract string, tokenId *big.Int) bool {
	instance := NewConnector(contract)

	if metaUrl, err := instance.nft.TokenURI(&bind.CallOpts{Context: context.Background()}, tokenId); err == nil {
		if metaUrl == "" || len(metaUrl) < 4 {
			return false
		}

		obj := Nft{}
		obj.Uri = metaUrl
		obj.Contract = contract
		obj.TokenId = tokenId.String()
		model := obj.toModel()
		nftChan <- &model
		if obj.Uri != "" {
			return true
		}
		//??????   ?????????url ????????????????????????

		if metaUrl[:4] == "ipfs" {
			//metaUrl = "https://ipfs.io/ipfs/" + metaUrl[7:]
			//ipfs?????????????????? ??????
			return false
		}
		jsonStr, err := Get(metaUrl)

		if err != nil {
			return false
			//log.Println("sync_process uploadNft Get err:", err)
		} else {
			obj := Nft{}
			err = json.Unmarshal([]byte(jsonStr), &obj)
			obj.Uri = metaUrl
			obj.Contract = contract
			obj.TokenId = tokenId.String()
			model := obj.toModel()
			nftChan <- &model
		}
	} else {
		return false
		//log.Println("sync_process uploadNft err:", err)
	}
	return true
}
func contractCall(addr, input string) string {
	data := common2.CallParamTemp{To: addr, Data: input}
	if ret, err := common2.ETHCall(data); err == nil {
		return ret
	} else {
		return ""
	}
}

func Post(data interface{}, api string) (string, error) {
	contentType := "application/json"
	client := &http.Client{Timeout: 3 * time.Second}
	jsonStr, _ := json.Marshal(data)
	resp, err := client.Post(postUrl+api, contentType, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("Post "+api+"  :", err)
		return "", err
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	return string(result), nil
}
func Get(url string) (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		//log.Println("Get "+url+"  :", err)
		return "", err
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	return string(result), nil
}
