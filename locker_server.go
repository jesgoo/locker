package main

import (
	"code.google.com/p/gcfg"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/locker/utils"
)

type GlobalData struct {
	Log struct {
		LogLevel   int
		LogCutTime int64
		LogDir     string
	}
	MySql struct {
		Location string
	}
	Server struct {
		Port int
	}
	LockerData struct {
		FirstAmount int
	}
}

var globalConf GlobalData

type CommonResp struct {
	Status uint32 `json:"status"`
}

type Message struct {
	Id    uint32 `json:"id"`
	Title string `json:"title"`
	Msg   string `json:"msg"`
}

type AmountResp struct {
	Amount uint32 `json:"amount"`
	Total  uint32 `json:"total"`
}
type GetMoney struct {
	Uid    string
	Method int
	Amount int
}

func InitServer() (err error) {
	err = gcfg.ReadFileInto(&globalConf, "./conf/locker.conf")
	if err != nil {
		log.Fatalf("Init server fail .err [%s]\n", err.Error())
		return
	}

	/************ init log **************/
	switch globalConf.Log.LogLevel {
	case 1:
		utils.GlobalLogLevel = utils.NoticeLevel
	case 2:
		utils.GlobalLogLevel = utils.FatalLevel
	case 3:
		utils.GlobalLogLevel = utils.WarningLevel
	case 4:
		utils.GlobalLogLevel = utils.DebugLevel
	default:
		utils.GlobalLogLevel = utils.DebugLevel
	}
	//	utils.GlobalLogLevel = utils.WarningLevel
	utils.DebugLog = &utils.LogControl{}
	utils.FatalLog = &utils.LogControl{}
	utils.WarningLog = &utils.LogControl{}
	utils.NoticeLog = &utils.LogControl{}
	err = utils.DebugLog.Init(globalConf.Log.LogCutTime, "locker.dg", globalConf.Log.LogDir, utils.DebugLevel)
	if err != nil {
		return
	}
	err = utils.FatalLog.Init(globalConf.Log.LogCutTime, "locker.fatal", globalConf.Log.LogDir, utils.FatalLevel)
	if err != nil {
		return
	}
	err = utils.WarningLog.Init(globalConf.Log.LogCutTime, "locker.warn", globalConf.Log.LogDir, utils.WarningLevel)
	if err != nil {
		return
	}
	err = utils.NoticeLog.Init(globalConf.Log.LogCutTime, "locker.log", globalConf.Log.LogDir, utils.NoticeLevel)
	if err != nil {
		return
	}
	/*********** init log end ***********/

	utils.DbClient = &utils.DbClientStruct{}
	utils.DbClient.Init(globalConf.MySql.Location, globalConf.LockerData.FirstAmount)
	return
}

func SearchMoney(resp http.ResponseWriter, req *http.Request) {

	err := req.ParseForm()
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	uidstr := req.Form["uid"]
	if len(uidstr) != 1 {
		utils.WarningLog.Write("no uid in request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(uidstr[0]) != 11 {
		utils.WarningLog.Write("uid len is not 11 . len[%d]", len(uidstr[0]))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	uid, err := strconv.ParseUint(uidstr[0], 10, 64)
	if err != nil {
		utils.WarningLog.Write("parse uid str to int64 fail . err[%s]", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	var amountret AmountResp
	amountret.Amount, amountret.Total, err = utils.DbClient.SearchMoney(uid)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	ret, err := json.Marshal(amountret)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	utils.DebugLog.Write("response [%s]", amountret)
	resp.WriteHeader(http.StatusOK)
	resp.Write(ret)
	return
}

func SearchExchange(resp http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
	}
	uidstr := req.Form["uid"]
	if len(uidstr) != 1 {
		utils.WarningLog.Write("no uid in request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(uidstr[0]) != 11 {
		utils.WarningLog.Write("uid len is not 11 . len[%d]", len(uidstr[0]))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	uid, err := strconv.ParseUint(uidstr[0], 10, 64)
	if err != nil {
		utils.WarningLog.Write("parse uid str to int64 fail . err[%s]", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	var ans []utils.ExchangeItem
	ans, err = utils.DbClient.SearchExchange(uid)
	if err != nil {
		utils.WarningLog.Write("dbclient search exchange fail . err[%s]", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	ret, err := json.Marshal(ans)
	resp.WriteHeader(http.StatusOK)
	resp.Write(ret)
	return
}

func SearchNews(resp http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	lastidstr := req.Form["lastid"]
	if len(lastidstr) != 1 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	idx, err := strconv.Atoi(lastidstr[0])
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	var msgret []Message
	var msgtmp Message
	msgtmp.Id = 1
	msgtmp.Title = "欢迎使用钱包锁屏"
	msgtmp.Msg = "欢迎使用钱包锁屏"
	msgret = append(msgret, msgtmp)
	if idx >= len(msgret) {
		resp.WriteHeader(http.StatusOK)
		return
	}
	res, err := json.Marshal(msgret)
	resp.WriteHeader(http.StatusOK)
	resp.Write(res)
	return
}
func AddExchange(resp http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		utils.WarningLog.Write("AddExchange reqbody is nil")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	reqdata, err := ioutil.ReadAll(req.Body)
	var tmpreq GetMoney
	err = json.Unmarshal(reqdata, &tmpreq)
	if err != nil {
		utils.WarningLog.Write("parse AddExchage request fail . err[%s]", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	var uid uint64
	var respx CommonResp
	if len(tmpreq.Uid) != 11 {
		utils.WarningLog.Write("uid len is not 11 . len[%d]", len(tmpreq.Uid))
		respx.Status = 400
		goto END
	}
	uid, err = strconv.ParseUint(tmpreq.Uid, 10, 64)
	if err != nil {
		utils.WarningLog.Write("parse uid str to int64 fail . err[%s]", err.Error())
		respx.Status = 400
		goto END
	}
	err = utils.DbClient.InsertExchange(uid, tmpreq.Method, tmpreq.Amount)
	if err != nil {
		utils.WarningLog.Write("add exchange fail . err[%s]", err.Error())
		respx.Status = 400
		goto END
	}
	respx.Status = 200
END:
	ret, err := json.Marshal(respx)
	if err != nil {
		utils.WarningLog.Write("json marshal fail. err[%s]", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.WriteHeader(http.StatusOK)
	resp.Write(ret)
	return
}

func AddUser(resp http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
	}
	var respx CommonResp
	var uid uint64
	var ret int
	uidstr := req.Form["uid"]
	if len(uidstr) != 1 {
		utils.WarningLog.Write("no uid in request")
		respx.Status = 400
		goto END
	}
	if len(uidstr[0]) != 11 {
		utils.WarningLog.Write("uid len is not 11 . len[%d]", len(uidstr[0]))
		respx.Status = 400
		goto END
	}
	uid, err = strconv.ParseUint(uidstr[0], 10, 64)
	if err != nil {
		utils.WarningLog.Write("parse uid str to int64 fail . err[%s]", err.Error())
		respx.Status = 400
		goto END
	}
	ret, err = utils.DbClient.AddUser(uid)
	if err != nil {
		utils.WarningLog.Write("dbclient add user fail . err[%s]", err.Error())
		respx.Status = 400
		goto END
	}
	switch ret {
	case 200:
		respx.Status = 200
	case 204:
		respx.Status = 204
	default:
		respx.Status = 400
	}
END:
	res, err := json.Marshal(respx)
	if err != nil {
		utils.WarningLog.Write("json marshal resp fail . err[%s]", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.WriteHeader(http.StatusOK)
	resp.Write(res)
	return
}
func main() {

	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.Println("start server")
	err := InitServer()
	if err != nil {
		log.Fatalf("Init server fail . err[%s]\n", err.Error())
		return
	}

	http.HandleFunc("/amount", SearchMoney)
	http.HandleFunc("/exchange", SearchExchange)
	http.HandleFunc("/news", SearchNews)
	http.HandleFunc("/getmoney", AddExchange)
	http.HandleFunc("/createuser", AddUser)
	listenstr := fmt.Sprintf(":%d", globalConf.Server.Port)
	err = http.ListenAndServe(listenstr, nil)
	if err != nil {
		log.Fatalf("start server fail . err[%s]\n", err.Error())
		return
	}
	return
}
