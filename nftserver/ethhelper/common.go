package ethhelper

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nftexchange/nftserver/ethhelper/common"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
)

var (
	client *rpc.Client
)

func init() {
	client, _ = rpc.Dial(common.MainPoint)
	if client == nil {
		log.Println("rpc.Dial err")
		return
	}
}

// GenCreateNftSign
// @description 生成创建Nft得签名
// @auth chen.gang 2021/9/30 15:00
// @param contract-合約地址 owner-nft得所有者 metaurl-metaurl   amount-数量   royalty-版税
// @return balance err
func GenCreateNftSign(contract,owner,metaUrl,tokenId,amount,royalty string ) (error,string) {
	jsFile, err := filepath.Abs("ethhelper/jsassist/gen_nft_sign.js")
	if err != nil {
		return err,""
	}
	cmd := exec.Command("node", jsFile, contract,owner,tokenId,amount,royalty,"")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return err,""
	}
	fmt.Println("GenCreateNftSign args:",cmd.Args)
	if err = cmd.Wait(); err != nil {
		return err,""
	}
	fmt.Println("GenCreateNftSign out:",out.String())
	return nil,out.String()
}

// BalanceOf
// @description 查询账户eth余额
// @auth chen.gang 2021/9/1 10:57
// @param address字符串
// @return balance err
func BalanceOf(addr string) (Balance int64, err error) {
	var balance string
	err = client.Call(&balance, "eth_getBalance", addr, "latest")
	if err != nil {
		return -1, err
	}
	Balance, _ = strconv.ParseInt(balance, 0, 64)
	return Balance, nil
}

// GetBlock
// @description 查询账户eth余额
// @auth chen.gang 2021/9/1 10:57
// @param address字符串
// @return balance err
func GetBlock(number string) (Balance Block, err error) {
	var block Block
	err = client.Call(&block, "eth_getBlockByNumber", number, true)
	if err != nil {
		return block, err
	}
	return block, nil
}

// TransactionCount
// @description 查询账户nonce
// @auth chen.gang 2021/9/1 10:57
// @param address字符串
// @return count err
func TransactionCount(addr string) (count int64, err error) {
	var c string
	err = client.Call(&c, "eth_getTransactionCount", addr, "latest")
	if err != nil {
		return -1, err
	}
	count, _ = strconv.ParseInt(c, 0, 64)
	return count, nil
}

// TransactionReceipt
// @description 查询Receipt
// @auth chen.gang 2021/9/1 10:57
// @param txHash 交易hash
// @return Receipt err
func TransactionReceipt(txHash string) (ret Receipt, err error) {
	err = client.Call(&ret, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return Receipt{}, err
	}
	return ret, nil
}

// ETHCall
// @description 发起合约call调用
// @auth chen.gang 2021/9/1 10:57
// @param params 入参
// @return ret err
func ETHCall(params CallParamTemp) (ret string, err error) {
	err = client.Call(&ret, "eth_call", params, "latest")
	if err != nil {
		return "", err
	}
	return ret, nil
}

// GetLogs
// @description 发起合约call调用
// @auth chen.gang 2021/9/1 10:57
// @param params 入参
// @return ret err
func GetLogs(filter LogFilter) (ret []Log, err error) {
	err = client.Call(&ret, "eth_getLogs", filter, "latest")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ValidateSign
// @description 验证签名是否正确 ，拍卖交易使用，验证买家出价情况
// @auth chen.gang 2021/9/1 10:57
// @param signHash 前端签名信息   originData 原始数据
// @return ret err
func ValidateSign(signHash, originData string) bool {
	return true
}

// SendRawTransaction
// @description 发送离线签名交易
// @auth chen.gang 2021/9/1 10:57
// @param rawTransaction tx签名data
// @return ret err
func SendRawTransaction(rawTransaction string) error {
	var ret string
	if err := client.Call(&ret, "eth_sendRawTransaction", rawTransaction); err != nil {
		return errors.New("Call failed:" + err.Error())
	}
	return nil
}
