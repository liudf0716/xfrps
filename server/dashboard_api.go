// Copyright 2017 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/KunTengRom/xfrps/models/config"
	"github.com/KunTengRom/xfrps/models/consts"
	"github.com/KunTengRom/xfrps/utils/log"
	"github.com/KunTengRom/xfrps/utils/util"
	"github.com/KunTengRom/xfrps/utils/version"

	"github.com/julienschmidt/httprouter"
)

var globalClientStats []*ClientStats
var globalProxyStats []*ProxyStats

type GeneralResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// api/serverinfo
type ServerInfoResp struct {
	GeneralResponse

	Version          string `json:"version"`
	VhostHttpPort    int64  `json:"vhost_http_port"`
	VhostHttpsPort   int64  `json:"vhost_https_port"`
	AuthTimeout      int64  `json:"auth_timeout"`
	SubdomainHost    string `json:"subdomain_host"`
	MaxPoolCount     int64  `json:"max_pool_count"`
	HeartBeatTimeout int64  `json:"heart_beat_timeout"`

	TotalTrafficIn  int64            `json:"total_traffic_in"`
	TotalTrafficOut int64            `json:"total_traffic_out"`
	CurConns        int64            `json:"cur_conns"`
	ClientCounts    int64            `json:"client_counts"`
	ProxyTypeCounts map[string]int64 `json:"proxy_type_count"`
}

func apiServerInfo(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res ServerInfoResp
	)
	defer func() {
		log.Info("Http response [/api/serverinfo]: code [%d]", res.Code)
	}()

	log.Info("Http request: [/api/serverinfo]")
	cfg := config.ServerCommonCfg
	serverStats := StatsGetServer()
	res = ServerInfoResp{
		Version:          version.Full(),
		VhostHttpPort:    cfg.VhostHttpPort,
		VhostHttpsPort:   cfg.VhostHttpsPort,
		AuthTimeout:      cfg.AuthTimeout,
		SubdomainHost:    cfg.SubDomainHost,
		MaxPoolCount:     cfg.MaxPoolCount,
		HeartBeatTimeout: cfg.HeartBeatTimeout,

		TotalTrafficIn:  serverStats.TotalTrafficIn,
		TotalTrafficOut: serverStats.TotalTrafficOut,
		CurConns:        serverStats.CurConns,
		ClientCounts:    serverStats.ClientCounts,
		ProxyTypeCounts: serverStats.ProxyTypeCounts,
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// Get client info
type ClientStatsInfo struct {
	RunId         string `json:"runid"`
	ProxyNum      int64  `json:"proxy_num"`
	ConnNum       int64  `json:"conn_num"`
	LastStartTime string `json:"last_start_time"`
	LastCloseTime string `json:"last_close_time"`
}

type GetClientInfoResp struct {
	GeneralResponse
	TotalPage	int64              	`json:"total_page"`
	TotalNum	int64				`json:"total_num"`
	Clients		[]*ClientStatsInfo 	`json:"clients"`
}

func apiClientByStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params, online int, pageSize int) {
	var (
		buf []byte
		res GetClientInfoResp
	)
	status := "online"
	if online == 0 {
		status = "offline"
	}
	defer func() {
		log.Info("Http response [/api/client/%s]: code [%d]", status, res.Code)
	}()

	pageNo := params.ByName("pageNo")
	pageIndex, err := strconv.Atoi(pageNo)
	if err != nil {
		log.Info("Http request: [/api/client/%s]", status)
		getAllClientStats(online)
		res.TotalPage 	= int64(len(globalClientStats)/pageSize + 1)
	} else {
		log.Info("Http request: [/api/client/%s/%d]", status, pageIndex)
		res.TotalPage = 0
		res.Clients = getClientStatsByPage(pageIndex, 100)
	}
	res.TotalNum 	= int64(len(globalClientStats))
	
	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// api/client/online
func apiClientOnline(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	apiClientByStatus(w, r, params, 1, 100)
}

// api/client/offline
func apiClientOffline(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	apiClientByStatus(w, r, params, 0, 100)
}

func getAllClientStats(online int) {
	globalClientStats = StatsGetClient(online)
}

func getClientStatsByPage(pageNo int, pageSize int) (clientInfos []*ClientStatsInfo) {
	clientInfos = make([]*ClientStatsInfo, 0, pageSize)
	start := pageNo * pageSize
	for i := start; i < len(globalClientStats) && i < start+pageSize; i++ {
		ps := globalClientStats[i]
		clientInfo := &ClientStatsInfo{}
		clientInfo.RunId = ps.RunId
		clientInfo.ProxyNum = ps.ProxyNum
		clientInfo.ConnNum = ps.ConnNum
		clientInfo.LastStartTime = ps.LastStartTime
		clientInfo.LastCloseTime = ps.LastCloseTime
		clientInfos = append(clientInfos, clientInfo)
	}
	return
}

// Get proxy info.
type ProxyStatsInfo struct {
	Name            string           `json:"name"`
	Conf            config.ProxyConf `json:"conf"`
	TodayTrafficIn  int64            `json:"today_traffic_in"`
	TodayTrafficOut int64            `json:"today_traffic_out"`
	CurConns        int64            `json:"cur_conns"`
	LastStartTime   string           `json:"last_start_time"`
	LastCloseTime   string           `json:"last_close_time"`
	Status          string           `json:"status"`
}

type GetProxyInfoResp struct {
	GeneralResponse
	TotalPage 	int64			`json:"total_page"`
	TotalNum	int64			`json:"total_num"`
	Proxies   	[]*ProxyStatsInfo `json:"proxies"`
}

func proxyOperation(w http.ResponseWriter, r *http.Request, params httprouter.Params, proxyType string, pageSize int) {
	var (
		buf []byte
		res GetProxyInfoResp
	)
	defer func() {
		log.Info("Http response [/api/proxy/%s]: code [%d]", proxyType, res.Code)
	}()

	pageNo := params.ByName("pageNo")
	pageIndex, err := strconv.Atoi(pageNo)
	if err != nil {
		log.Info("Http request: [/api/proxy/%s]", proxyType)
		getProxyStatsByType(proxyType)
		res.TotalPage = int64(len(globalProxyStats)/pageSize + 1)
	} else {
		log.Info("Http request: [/api/proxy/%s/%d]", proxyType, pageIndex)
		res.TotalPage = 0
		res.Proxies = getProxyStatsPageByType(proxyType, pageIndex, pageSize)
	}
	res.TotalNum 	= int64(len(globalProxyStats))
	
	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// api/proxy/tcp
func apiProxyTcp(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	proxyOperation(w, r, params, consts.TcpProxy, 100)
}

// api/proxy/udp
func apiProxyUdp(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	proxyOperation(w, r, params, consts.UdpProxy, 100)
}

// api/proxy/ftp
func apiProxyFtp(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	proxyOperation(w, r, params, consts.FtpProxy, 100)
}

// api/proxy/http
func apiProxyHttp(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	proxyOperation(w, r, params, consts.HttpProxy, 100)
}

// api/proxy/https
func apiProxyHttps(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	proxyOperation(w, r, params, consts.HttpsProxy, 100)
}

func getProxyStatsPageByType(proxyType string, pageNo int, pageSize int) (proxyInfos []*ProxyStatsInfo) {
	proxyInfos = make([]*ProxyStatsInfo, 0, pageSize)
	start := pageNo * pageSize
	for i := start; i < len(globalProxyStats) && i < start+pageSize; i++ {
		ps := globalProxyStats[i]
		proxyInfo := &ProxyStatsInfo{}
		if pxy, ok := ServerService.pxyManager.GetByName(ps.Name); ok {
			proxyInfo.Conf = pxy.GetConf()
			proxyInfo.Status = consts.Online
		} else {
			proxyInfo.Status = consts.Offline
		}
		proxyInfo.Name = ps.Name
		proxyInfo.TodayTrafficIn = ps.TodayTrafficIn
		proxyInfo.TodayTrafficOut = ps.TodayTrafficOut
		proxyInfo.CurConns = ps.CurConns
		proxyInfo.LastStartTime = ps.LastStartTime
		proxyInfo.LastCloseTime = ps.LastCloseTime
		proxyInfos = append(proxyInfos, proxyInfo)
	}
	return
}

func getProxyStatsByType(proxyType string) {
	globalProxyStats = StatsGetProxiesByType(proxyType)
}

// api/proxy/traffic/:name
type GetProxyTrafficResp struct {
	GeneralResponse

	Name       string  `json:"name"`
	TrafficIn  []int64 `json:"traffic_in"`
	TrafficOut []int64 `json:"traffic_out"`
}

func apiProxyTraffic(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res GetProxyTrafficResp
	)
	name := params.ByName("name")

	defer func() {
		log.Info("Http response [/api/proxy/traffic/:name]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/proxy/traffic/:name]")

	res.Name = name
	proxyTrafficInfo := StatsGetProxyTraffic(name)
	if proxyTrafficInfo == nil {
		res.Code = 1
		res.Msg = "no proxy info found"
	} else {
		res.TrafficIn = proxyTrafficInfo.TrafficIn
		res.TrafficOut = proxyTrafficInfo.TrafficOut
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// /api/port/getfree/:proto
type GetFreePortResp struct {
	GeneralResponse

	Proto    string `json:"proto"`
	FreePort int    `json:"free_port"`
}

func apiGetFreePort(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res GetFreePortResp
	)

	proto := params.ByName("proto")
	defer func() {
		log.Info("Http response [/api/port/getfree/:proto]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/port/getfree/:proto]")

	res.Proto = proto
	if proto == "tcp" {
		freePort := util.RandomTCPPort()
		if freePort > 0 {
			res.FreePort = freePort
		} else {
			res.Code = 1
			res.Msg = "no free tcp port"
		}
	} else {
		res.Code = 1
		res.Msg = "not support proto " + proto
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// /api/port/tcp/isfree/:port

func apiIsTcpPortFree(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res GeneralResponse
	)

	strPort := params.ByName("port")
	defer func() {
		log.Info("Http response [/api/port/tcp/isfree/:port]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/port/tcp/isfree/:port]")

	port, err := strconv.Atoi(strPort)
	if err == nil {
		res.Code = 1
		res.Msg = "port is illegal"
	} else if util.IsTCPPortAvailable(port) {
		res.Code = 0
		res.Msg = "tcp port available"
	} else {
		res.Code = 1
		res.Msg = "tcp port unavailable"
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

type GetPortResp struct {
	GeneralResponse

	Port int64 `json:"port"`
}

// /api/port/tcp/getport/:runid

func apiGetPort(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res GetPortResp
	)

	runid := params.ByName("runid")
	defer func() {
		log.Info("Http response [/api/port/tcp/getport/:runid]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/port/tcp/getport/:runid]")

	port, ok := ServerService.portManager.GetById(runid)
	if ok {
		res.Port = port
	} else {
		res.Code = 1
		res.Msg = "can not get port by its runid"
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// /api/port/tcp/getftpport/:runid
func apiGetFtpPort(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res GetPortResp
	)

	runid := params.ByName("runid")
	defer func() {
		log.Info("Http response [/api/port/tcp/getftpport/:runid]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/port/tcp/getftpport/:runid]")

	port, ok := ServerService.portManager.GetFtpById(runid)
	if ok {
		res.Port = port
	} else {
		res.Code = 1
		res.Msg = "can not get ftp control port by its runid"
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}
