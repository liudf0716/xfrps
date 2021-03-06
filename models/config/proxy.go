// Copyright 2016 fatedier, fatedier@gmail.com
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

package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/liudf0716/xfrps/models/consts"
	"github.com/liudf0716/xfrps/models/msg"

	"github.com/liudf0716/xfrps/utils/util"
	ini "github.com/vaughan0/go-ini"
)

var proxyConfTypeMap map[string]reflect.Type

func init() {
	proxyConfTypeMap = make(map[string]reflect.Type)
	proxyConfTypeMap[consts.TcpProxy] = reflect.TypeOf(TcpProxyConf{})
	proxyConfTypeMap[consts.UdpProxy] = reflect.TypeOf(UdpProxyConf{})
	proxyConfTypeMap[consts.HttpProxy] = reflect.TypeOf(HttpProxyConf{})
	proxyConfTypeMap[consts.HttpsProxy] = reflect.TypeOf(HttpsProxyConf{})
	proxyConfTypeMap[consts.FtpProxy] = reflect.TypeOf(FtpProxyConf{})
}

// NewConfByType creates a empty ProxyConf object by proxyType.
// If proxyType isn't exist, return nil.
func NewConfByType(proxyType string) ProxyConf {
	v, ok := proxyConfTypeMap[proxyType]
	if !ok {
		return nil
	}
	cfg := reflect.New(v).Interface().(ProxyConf)
	return cfg
}

type ProxyConf interface {
	GetName() string
	GetBaseInfo() *BaseProxyConf
	LoadFromMsg(pMsg *msg.NewProxy)
	LoadFromFile(name string, conf ini.Section) error
	UnMarshalToMsg(pMsg *msg.NewProxy)
	Check() error
	FillLocalServer(ip string, port int)
	FillRemotePort(rport int64)
}

func NewProxyConf(pMsg *msg.NewProxy) (cfg ProxyConf, err error) {
	if pMsg.ProxyType == "" {
		pMsg.ProxyType = consts.TcpProxy
	}

	cfg = NewConfByType(pMsg.ProxyType)
	if cfg == nil {
		err = fmt.Errorf("proxy [%s] type [%s] error", pMsg.ProxyName, pMsg.ProxyType)
		return
	}
	cfg.LoadFromMsg(pMsg)
	err = cfg.Check()
	return
}

func NewProxyConfFromFile(name string, section ini.Section) (cfg ProxyConf, err error) {
	proxyType := section["type"]
	if proxyType == "" {
		proxyType = consts.TcpProxy
		section["type"] = consts.TcpProxy
	}
	cfg = NewConfByType(proxyType)
	if cfg == nil {
		err = fmt.Errorf("proxy [%s] type [%s] error", name, proxyType)
		return
	}
	err = cfg.LoadFromFile(name, section)
	return
}

// BaseProxy info
type BaseProxyConf struct {
	ProxyName string `json:"proxy_name"`
	ProxyType string `json:"proxy_type"`

	UseEncryption  bool `json:"use_encryption"`
	UseCompression bool `json:"use_compression"`
}

func (cfg *BaseProxyConf) GetName() string {
	return cfg.ProxyName
}

func (cfg *BaseProxyConf) GetBaseInfo() *BaseProxyConf {
	return cfg
}

func (cfg *BaseProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.ProxyName = pMsg.ProxyName
	cfg.ProxyType = pMsg.ProxyType
	cfg.UseEncryption = pMsg.UseEncryption
	cfg.UseCompression = pMsg.UseCompression
}

func (cfg *BaseProxyConf) LoadFromFile(name string, section ini.Section) error {
	var (
		tmpStr string
		ok     bool
	)
	if ClientCommonCfg.User != "" {
		cfg.ProxyName = ClientCommonCfg.User + "." + name
	} else {
		cfg.ProxyName = name
	}
	cfg.ProxyType = section["type"]

	tmpStr, ok = section["use_encryption"]
	if ok && tmpStr == "true" {
		cfg.UseEncryption = true
	}

	tmpStr, ok = section["use_compression"]
	if ok && tmpStr == "true" {
		cfg.UseCompression = true
	}
	return nil
}

func (cfg *BaseProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	pMsg.ProxyName = cfg.ProxyName
	pMsg.ProxyType = cfg.ProxyType
	pMsg.UseEncryption = cfg.UseEncryption
	pMsg.UseCompression = cfg.UseCompression
}

// Bind info
// local service port map to remote remote port
type BindInfoConf struct {
	BindAddr   string `json:"bind_addr"`
	RemotePort int64  `json:"remote_port"`
}

func (cfg *BindInfoConf) LoadFromMsg(pMsg *msg.NewProxy) {
	if ServerCommonCfg != nil {
		cfg.BindAddr = ServerCommonCfg.BindAddr
	}

	cfg.RemotePort = pMsg.RemotePort
}

func (cfg *BindInfoConf) LoadFromFile(name string, section ini.Section) (err error) {
	var (
		tmpStr string
		ok     bool
	)
	if tmpStr, ok = section["remote_port"]; ok {
		if cfg.RemotePort, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] remote_port error", name)
		}
	} else {
		cfg.RemotePort = 0
	}
	return nil
}

func (cfg *BindInfoConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	pMsg.RemotePort = cfg.RemotePort
}

func (cfg *BindInfoConf) check() (err error) {

	if ServerCommonCfg != nil && cfg.RemotePort != 0 && len(ServerCommonCfg.PrivilegeAllowPorts) != 0 {
		if ok := util.ContainsPort(ServerCommonCfg.PrivilegeAllowPorts, cfg.RemotePort); !ok {
			return fmt.Errorf("remote port [%d] isn't allowed", cfg.RemotePort)
		}
	}

	if cfg.RemotePort != 0 && !util.IsTCPPortAvailable(int(cfg.RemotePort)) {
		return fmt.Errorf("remote port [%d] isn't available", cfg.RemotePort)
	}

	return nil
}

func (cfg *BindInfoConf) FillRemotePort(rport int64) {
	cfg.RemotePort = rport
}

// Domain info
type DomainConf struct {
	CustomDomains []string `json:"custom_domains"`
	SubDomain     string   `json:"sub_domain"`
}

func (cfg *DomainConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.CustomDomains = pMsg.CustomDomains
	cfg.SubDomain = pMsg.SubDomain
}

func (cfg *DomainConf) LoadFromFile(name string, section ini.Section) (err error) {
	var (
		tmpStr string
		ok     bool
	)
	if tmpStr, ok = section["custom_domains"]; ok {
		cfg.CustomDomains = strings.Split(tmpStr, ",")
		for i, domain := range cfg.CustomDomains {
			cfg.CustomDomains[i] = strings.ToLower(strings.TrimSpace(domain))
		}
	}

	if tmpStr, ok = section["subdomain"]; ok {
		cfg.SubDomain = tmpStr
	}

	if len(cfg.CustomDomains) == 0 && cfg.SubDomain == "" {
		return fmt.Errorf("Parse conf error: proxy [%s] custom_domains and subdomain should set at least one of them", name)
	}
	return
}

func (cfg *DomainConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	pMsg.CustomDomains = cfg.CustomDomains
	pMsg.SubDomain = cfg.SubDomain
}

func (cfg *DomainConf) check() (err error) {
	for _, domain := range cfg.CustomDomains {
		if ServerCommonCfg.SubDomainHost != "" && len(strings.Split(ServerCommonCfg.SubDomainHost, ".")) < len(strings.Split(domain, ".")) {
			if strings.Contains(domain, ServerCommonCfg.SubDomainHost) {
				return fmt.Errorf("custom domain [%s] should not belong to subdomain_host [%s]", domain, ServerCommonCfg.SubDomainHost)
			}
		}
	}

	if cfg.SubDomain != "" {
		if ServerCommonCfg.SubDomainHost == "" {
			return fmt.Errorf("subdomain is not supported because this feature is not enabled by frps")
		}
		if strings.Contains(cfg.SubDomain, ".") || strings.Contains(cfg.SubDomain, "*") {
			return fmt.Errorf("'.' and '*' is not supported in subdomain")
		}
	}
	return nil
}

// Local service info
type LocalSvrConf struct {
	LocalIp   string `json:"-"`
	LocalPort int    `json:"-"`
}

func (cfg *LocalSvrConf) setLocalServer(ip string, port int) {
	cfg.LocalIp = ip
	cfg.LocalPort = port
}

func (cfg *LocalSvrConf) LoadFromFile(name string, section ini.Section) (err error) {
	if cfg.LocalIp = section["local_ip"]; cfg.LocalIp == "" {
		cfg.LocalIp = "127.0.0.1"
	}

	if tmpStr, ok := section["local_port"]; ok {
		if cfg.LocalPort, err = strconv.Atoi(tmpStr); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] local_port error", name)
		}
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] local_port not found", name)
	}
	return nil
}

type PluginConf struct {
	Plugin       string            `json:"-"`
	PluginParams map[string]string `json:"-"`
}

func (cfg *PluginConf) LoadFromFile(name string, section ini.Section) (err error) {
	cfg.Plugin = section["plugin"]
	cfg.PluginParams = make(map[string]string)
	if cfg.Plugin != "" {
		// get params begin with "plugin_"
		for k, v := range section {
			if strings.HasPrefix(k, "plugin_") {
				cfg.PluginParams[k] = v
			}
		}
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] no plugin info found", name)
	}
	return
}

// TCP
type TcpProxyConf struct {
	BaseProxyConf
	BindInfoConf

	LocalSvrConf
	PluginConf

	FtpCfgProxyName string `json:"-"`
}

func (cfg *TcpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.BindInfoConf.LoadFromMsg(pMsg)
	cfg.FtpCfgProxyName = pMsg.FtpCfgProxyName
}

func (cfg *TcpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.BindInfoConf.LoadFromFile(name, section); err != nil {
		return
	}

	if err = cfg.PluginConf.LoadFromFile(name, section); err != nil {
		if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
			return
		}
	}
	return
}

func (cfg *TcpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.BindInfoConf.UnMarshalToMsg(pMsg)
}

func (cfg *TcpProxyConf) Check() (err error) {
	err = cfg.BindInfoConf.check()
	return
}

func (cfg *TcpProxyConf) FillLocalServer(ip string, port int) {
	cfg.LocalSvrConf.setLocalServer(ip, port)
}

func (cfg *TcpProxyConf) FillRemotePort(rport int64) {
	cfg.BindInfoConf.FillRemotePort(rport)
}

// UDP
type UdpProxyConf struct {
	BaseProxyConf
	BindInfoConf

	LocalSvrConf
}

func (cfg *UdpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.BindInfoConf.LoadFromMsg(pMsg)
}

func (cfg *UdpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.BindInfoConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
		return
	}
	return
}

func (cfg *UdpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.BindInfoConf.UnMarshalToMsg(pMsg)
}

func (cfg *UdpProxyConf) Check() (err error) {
	err = cfg.BindInfoConf.check()
	return
}

func (cfg *UdpProxyConf) FillLocalServer(ip string, port int) {
	cfg.LocalSvrConf.setLocalServer(ip, port)
}

func (cfg *UdpProxyConf) FillRemotePort(rport int64) {
	cfg.BindInfoConf.FillRemotePort(rport)
}

// ftp
type FtpProxyConf struct {
	BaseProxyConf
	LocalSvrConf

	RemotePort     int64 `json:"remote_port"`
	RemoteDataPort int64 `json:"remote_data_port"`
}

func (cfg *FtpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.RemotePort = pMsg.RemotePort
	cfg.RemoteDataPort = pMsg.RemoteDataPort
}

func (cfg *FtpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}

	if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
		return
	}

	var (
		tmpStr string
		ok     bool
	)
	if tmpStr, ok = section["remote_port"]; ok {
		if cfg.RemotePort, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] remote_port error", name)
		}
	} else {
		cfg.RemotePort = 0
	}

	if tmpStr, ok = section["remote_data_port"]; ok {
		if cfg.RemoteDataPort, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] remote_data_port error", name)
		}
	} else {
		cfg.RemoteDataPort = 0
	}
	return
}

func (cfg *FtpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	pMsg.RemotePort = cfg.RemotePort
	pMsg.RemoteDataPort = cfg.RemoteDataPort
}

func (cfg *FtpProxyConf) Check() (err error) {
	return
}

func (cfg *FtpProxyConf) FillLocalServer(ip string, port int) {
	cfg.LocalSvrConf.setLocalServer(ip, port)
}

func (cfg *FtpProxyConf) FillRemotePort(rport int64) {
	cfg.RemotePort = rport
}

// func (cfg *FtpProxyConf) FillRemoteDataPort(rport int64) {
// 	cfg.RemoteDataPort = rport
// }

// HTTP
type HttpProxyConf struct {
	BaseProxyConf
	DomainConf

	LocalSvrConf
	PluginConf

	Locations         []string `json:"locations"`
	HostHeaderRewrite string   `json:"host_header_rewrite"`
	HttpUser          string   `json:"-"`
	HttpPwd           string   `json:"-"`
}

func (cfg *HttpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.DomainConf.LoadFromMsg(pMsg)

	cfg.Locations = pMsg.Locations
	cfg.HostHeaderRewrite = pMsg.HostHeaderRewrite
	cfg.HttpUser = pMsg.HttpUser
	cfg.HttpPwd = pMsg.HttpPwd
}

func (cfg *HttpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.DomainConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
		return
	}

	var (
		tmpStr string
		ok     bool
	)
	if tmpStr, ok = section["locations"]; ok {
		cfg.Locations = strings.Split(tmpStr, ",")
	} else {
		cfg.Locations = []string{""}
	}

	cfg.HostHeaderRewrite = section["host_header_rewrite"]
	cfg.HttpUser = section["http_user"]
	cfg.HttpPwd = section["http_pwd"]
	return
}

func (cfg *HttpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.DomainConf.UnMarshalToMsg(pMsg)

	pMsg.Locations = cfg.Locations
	pMsg.HostHeaderRewrite = cfg.HostHeaderRewrite
	pMsg.HttpUser = cfg.HttpUser
	pMsg.HttpPwd = cfg.HttpPwd
}

func (cfg *HttpProxyConf) Check() (err error) {
	if ServerCommonCfg.VhostHttpPort == 0 {
		return fmt.Errorf("type [http] not support when vhost_http_port is not set")
	}
	err = cfg.DomainConf.check()
	return
}

func (cfg *HttpProxyConf) FillLocalServer(ip string, port int) {
	cfg.LocalSvrConf.setLocalServer(ip, port)
}

func (cfg *HttpProxyConf) FillRemotePort(rport int64) {
	return
}

// HTTPS
type HttpsProxyConf struct {
	BaseProxyConf
	DomainConf

	LocalSvrConf
	PluginConf
}

func (cfg *HttpsProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.DomainConf.LoadFromMsg(pMsg)
}

func (cfg *HttpsProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.DomainConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
		return
	}
	return
}

func (cfg *HttpsProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.DomainConf.UnMarshalToMsg(pMsg)
}

func (cfg *HttpsProxyConf) Check() (err error) {
	if ServerCommonCfg.VhostHttpsPort == 0 {
		return fmt.Errorf("type [https] not support when vhost_https_port is not set")
	}
	err = cfg.DomainConf.check()
	return
}

func (cfg *HttpsProxyConf) FillLocalServer(ip string, port int) {
	cfg.LocalSvrConf.setLocalServer(ip, port)
}

func (cfg *HttpsProxyConf) FillRemotePort(rport int64) {
	return
}

// if len(startProxy) is 0, start all
// otherwise just start proxies in startProxy map
func LoadProxyConfFromFile(prefix string, conf ini.File, startProxy map[string]struct{}) (proxyConfs map[string]ProxyConf, err error) {
	if prefix != "" {
		prefix += "."
	}

	startAll := true
	if len(startProxy) > 0 {
		startAll = false
	}
	proxyConfs = make(map[string]ProxyConf)
	for name, section := range conf {
		_, shouldStart := startProxy[name]
		if name != "common" && (startAll || shouldStart) {
			cfg, err := NewProxyConfFromFile(name, section)
			if err != nil {
				return proxyConfs, err
			}
			proxyConfs[prefix+name] = cfg

			basePxyConf := cfg.GetBaseInfo()
			if basePxyConf.ProxyType == consts.FtpProxy {
				var msg msg.NewProxy
				cfg.UnMarshalToMsg(&msg)
				msg.FtpCfgProxyName = msg.ProxyName
				msg.ProxyName = fmt.Sprintf("%s_ftp_data_proxy", msg.ProxyName)
				msg.ProxyType = consts.TcpProxy
				msg.RemotePort = msg.RemoteDataPort

				ncfg, err1 := NewProxyConf(&msg)
				if err1 != nil {
					return proxyConfs, err1
				}
				proxyConfs[msg.ProxyName] = ncfg
			}
		}
	}
	return
}
