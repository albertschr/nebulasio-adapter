/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package nebulasio

import (
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/imroc/req"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
	"math/big"
	"strconv"

	//	"log"
	"errors"
)

type Client struct {
	BaseURL     string
	Debug       bool
	Client      *req.Req
	Header      req.Header
}

//定义全局变量Nonce用于记录真正交易上链的nonce值和记录在DB中的nonce值
var Nonce_Chain int

func NewClient(url string, debug bool) *Client {
	c := Client{
		BaseURL:     url,
		Debug:       debug,
	}

	api := req.New()

	c.Client = api
	c.Header = req.Header{"Content-Type": "application/json"}

	return &c
}

//func (c *Client) CallTestJson() (){
//
//	trx := make(map[string]interface{},0)
//
//	var Nonce string = "100"
//	nonce,_:= strconv.ParseUint(Nonce,10,64)
//
//	trx["from"] = "qwerty"
//	trx["to"] = "asdf"
//	trx["value"] = "123"
//	trx["nonce"] = nonce
//	trx["gasLimit"] = "1212"
//	trx["gasPrice"] = "10000"
//
//	fmt.Printf("trx=%v\n\n",trx)
//
//	tx := &rpcpb.TransactionRequest{
//		"qwerty",
//		"asdf",
//		"123",
//		123 ,
//		"123" ,
//		"1222",
//		nil,
//		nil,
//		nil,
//		"",
//	}
//	fmt.Printf("tx=%v\n",tx)
//}

//ConvertToBigInt  string转Big int
func ConvertToBigInt(value string) (*big.Int, error) {
	bigvalue := new(big.Int)
	var success bool

	_, success = bigvalue.SetString(value, 10)
	if !success {
		errInfo := fmt.Sprintf("convert value [%v] to bigint failed, check the value and base passed through\n", value)
		log.Errorf(errInfo)
		return big.NewInt(0), errors.New(errInfo)
	}
	return bigvalue, nil
}

//ConverWeiStringToNasDecimal 字符串Wei转小数NAS
func ConverWeiStringToNasDecimal(amount string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(amount)
	if err != nil {
		log.Error("convert string to deciaml failed, err=", err)
		return d, err
	}

	d = d.Div(coinDecimal)
	return d, nil
}

//ConvertNasStringToWei 字符串NAS转Wei
func ConvertNasStringToWei(amount string) (*big.Int, error) {
	//log.Debug("amount:", amount)
	vDecimal, _ := decimal.NewFromString(amount)
	//if err != nil {
	//	log.Error("convert from string to decimal failed, err=", err)
	//	return nil, err
	//}

	vDecimal = vDecimal.Mul(coinDecimal)
	rst := new(big.Int)
	if _, valid := rst.SetString(vDecimal.String(), 10); !valid {
		log.Error("conver to big.int failed")
		return nil, errors.New("conver to big.int failed")
	}
	return rst, nil
}

//确定nonce值
func (c *Client) CheckNonce(key *Key) uint64{

	nonce_get,_ := c.CallGetaccountstate(key.Address,"nonce")
	nonce_chain ,_ := strconv.Atoi(nonce_get) 	//当前链上nonce值
	nonce_db,_ := strconv.Atoi(key.Nonce)	//本地记录的nonce值

	//如果本地nonce_local > 链上nonce,采用本地nonce,否则采用链上nonce
	if nonce_db > nonce_chain{
		Nonce_Chain = nonce_db + 1
		//log.Std.Info("%s nonce_db=%d > nonce_chain=%d,Use nonce_db+1...",key.Address,nonce_db,nonce_chain)
	}else{
		Nonce_Chain = nonce_chain + 1
		//log.Std.Info("%s nonce_db=%d <= nonce_chain=%d,Use nonce_chain+1...",key.Address,nonce_db,nonce_chain)
	}

	return uint64(Nonce_Chain)
}

//查询每个地址balance、nonce
//address:n1S8ojaa9Pz8TduXEm8vXrxBs6Kz5dyp7km
//query:balance、nonce
func (c *Client) CallGetaccountstate( address string ,query string) (string, error) {

	url := c.BaseURL + "/v1/user/accountstate"

	if c.Debug {
		log.Info("URL :%v",url)
	}

	var (
		body = make(map[string]interface{}, 0)
	)

	if c.Client == nil {
		return "", errors.New("API url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "application/json",
		"Authorization": "Basic " ,
	}

	//json-rpc
//	body["jsonrpc"] = "2.0"
//	body["id"] = "1"
//	body["method"] = 10
//	body["params"] = 12121
	body["address"] = address

	if c.Debug {
		log.Info("Start Request API...")
	}

	r, err := c.Client.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return "", err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return "", err
	}
	//resp :  {"result":{"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}}
	//result:  "result" : {"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}
	//result:  "result.address" : "n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"
	dst := "result." + query
	result := resp.Get(dst)

	return result.Str, nil
}


//查询区块链chain_id，testnet:	mainnet:
func (c *Client) CallGetnebstate( query string) (*gjson.Result, error) {
	url := c.BaseURL + "/v1/user/nebstate"
	param := make(req.QueryParam, 0)

	r, err := c.Client.Get(url, param)
	if err != nil {
		log.Info(err)
		return nil,err
	}

	//	return r.Bytes()
	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return nil,err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil,err
	}

//	result := resp.Get("result.chain_id")

	dst := "result." + query
	result := resp.Get(dst)

	return &result, nil
}


//查询GasPrice
func (c *Client) CallGetGasPrice() string {
	url := c.BaseURL + "/v1/user/getGasPrice"
	param := make(req.QueryParam, 0)

	r, err := c.Client.Get(url, param)
	if err != nil {
		log.Info(err)
		return ""
	}

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return ""
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return ""
	}

	result := resp.Get("result.gas_price")
	return (result.Str)
}


//发送广播签名后的交易单数据
func (c *Client) CallSendRawTransaction( data string ) (string, error) {

	url := c.BaseURL + "/v1/user/rawtransaction"

	var (
		body = make(map[string]interface{}, 0)
	)

	if c.Client == nil {
		return "", errors.New("API url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "application/json",
		"Authorization": "Basic " ,
	}

	//json-rpc
	//	body["jsonrpc"] = "2.0"
	//	body["id"] = "1"
	//	body["method"] = path
	//	body["params"] = request
	body["data"] = data

	if c.Debug {
		log.Info("Start Request API...")
	}

	r, err := c.Client.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return "", err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return "", err
	}
	//resp :  {"result":{"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}}
	//result:  "result" : {"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}
	//result:  "result.address" : "n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"

	result := resp.Get("result.txhash")
	return result.Str, nil
}

//isError 是否报错
func isError(result *gjson.Result) error {

	if result.Get("error").Exists() {

		return errors.New(result.Get("error").String())
	}

	return nil
}

//根据区块高度获取区块信息
//
func (c *Client) CallgetBlockByHeightOrHash( input string ,heightOrhash int) (*gjson.Result, error) {

	var url_index string
	var (
		body = make(map[string]interface{}, 0)
	)
	body["full_fill_transaction"] = true

	if heightOrhash == byHeight{
		url_index = "/v1/user/getBlockByHeight"
		body["height"] = decimal.RequireFromString(input).IntPart()
	}else if heightOrhash == byHash{
		url_index = "/v1/user/getBlockByHash"
		body["hash"] = input
	}else{
		return nil, errors.New("input value invalid,please check !")
	}

	url := c.BaseURL + url_index

	if c.Client == nil {
		return nil, errors.New("API url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "application/json",
		"Authorization": "Basic " ,
	}

	if c.Debug {
		log.Info("Start Request API...")
	}

	r, err := c.Client.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil, err
	}
	/*
	{
    "result": {
        "hash": "95480cc637d0782c60f321b3600200074f468444c1399ae7bba0fc0f8007a410",
        "parent_hash": "59f927c87d5d4ca6f7d3c2827c42f8ec60f0057146ae371cdfa1fba8d0514f5e",
        "height": "8989",
        "nonce": "0",
        "coinbase": "n1NM2eETQG5Es7cCc7sh29NJr9cP94QZcXR",
        "timestamp": "1539161640",
        "chain_id": 100,
        "state_root": "39643466944ad6d31c9ffe9df8ae4d30b29abed91a285d293711fa548c4930ba",
        "txs_root": "702ff2561aead08ac7eb64e1aea5845d4517329a74738c3c830fe2670ee4c9ea",
        "events_root": "57aebd702400deec492a455144011f8abe42355a42f4323e380194b088363a16",
        "consensus_root": {
            "timestamp": "1539161640",
            "proposer": "GVcH+WT/SVMkY18ix7SG4F1+Z8evXJoA35c=",
            "dynasty_root": "GZDY8fY8Utgqftr+PUdJgtP82AybM9+4H6UFvJf/jAg="
        },
        "miner": "n1FF1nz6tarkDVwWQkMnnwFPuPKUaQTdptE",
        "randomSeed": "f13e03ea259581ef5c93353b8ee34cdbefc387466fa343cb27f088506ac93d07",
        "randomProof": "cf8f6f1f0d4ec7560eb9640da06989ed1849edcf1e3a167f58870594a087939e1bfe08c6419b316add5cd11b8a5b491415cc1e62dc0cb6b85f8096f3792b3cfc0425fa00842ca9e00558944f3797b42e4fea8b9d5dea4a6743b72e0fedf6633cdfa10f52f65b552668c3fae6d9da7df0d306841c2dbe03c01027fd63bc64fd7e8e",
        "is_finality": false,
        "transactions": []
    	}
	}
	*/
	//返回整个数据
	result := resp.Get("result")
	return &result, nil
}


func (c *Client) CallGetTransactionReceipt( txid string ) (*gjson.Result, error)  {

	url := c.BaseURL + "/v1/user/getTransactionReceipt"

	var (
		body = make(map[string]interface{}, 0)
	)

	if c.Client == nil {
		return nil, errors.New("API url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "application/json",
		"Authorization": "Basic " ,
	}

	//json-rpc
	//	body["jsonrpc"] = "2.0"
	//	body["id"] = "1"
	//	body["method"] = path
	//	body["params"] = request
	body["hash"] = txid

	if c.Debug {
		log.Info("Start Request API...")
	}

	r, err := c.Client.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil, err
	}
	//resp :  {"result":{"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}}
	//result:  "result" : {"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}
	//result:  "result.address" : "n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"

	result := resp.Get("result")
	return &result, nil
}

//由于接口是keep-alive connection，目前暂不支持
func (c *Client) CallGetsubscribe() (*gjson.Result, error)  {

	url := c.BaseURL + "/v1/user/subscribe"

	body := map[string]interface{}{

		"topics": []string{"chain.linkBlock","chain.pendingTransaction"},
	}

	if c.Client == nil {
		return nil, errors.New("API url is not setup. ")
	}

	authHeader := req.Header{
	//	"Connection": 	 "keep-alive",
		"Accept":        "application/json",
		"Authorization": "Basic " ,
	}

	fmt.Printf("url=%v,body=%v\n",url,body)

	if c.Debug {
		log.Info("Start Request API...")
	}

	r, err := c.Client.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil, err
	}

	result := resp.Get("result")
	return &result, nil
}

//估算交易gas用量
func (c *Client) CallGetestimateGas( parameter * estimateGasParameter ) (*gjson.Result, error)  {

	url := c.BaseURL + "/v1/user/estimateGas"

	var (
		body = make(map[string]interface{}, 0)
	)

	if c.Client == nil {
		return nil, errors.New("API url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "application/json",
		"Authorization": "Basic " ,
	}

	//json-rpc
	//	body["jsonrpc"] = "2.0"
	//	body["id"] = "1"
	//	body["method"] = path
	//	body["params"] = request
	body["from"] = parameter.from
	body["to"] = parameter.to
	body["value"] = parameter.value
	body["nonce"] = parameter.nonce
	body["gasPrice"] = parameter.gasPrice
	body["gasLimit"] = parameter.gasLimit


	if c.Debug {
		log.Info("Start Request API...")
	}

	r, err := c.Client.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil, err
	}
	//resp :  {"result":{"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}}
	//result:  "result" : {"address":"n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"}
	//result:  "result.address" : "n1Qmnmuebg4xxvnuHUoSLDjLFMznxMdsDng"

	result := resp.Get("result")
	return &result, nil
}


