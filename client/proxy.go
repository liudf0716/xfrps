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

package client

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"strconv"
	"strings"
	
	"github.com/KunTengRom/xfrps/models/config"
	"github.com/KunTengRom/xfrps/models/consts"
	"github.com/KunTengRom/xfrps/models/msg"
	"github.com/KunTengRom/xfrps/models/plugin"
	"github.com/KunTengRom/xfrps/models/proto/tcp"
	"github.com/KunTengRom/xfrps/models/proto/udp"
	"github.com/KunTengRom/xfrps/utils/errors"
	"github.com/KunTengRom/xfrps/utils/log"
	frpNet "github.com/KunTengRom/xfrps/utils/net"
)

// Proxy defines how to work for different proxy type.
type Proxy interface {
	Run() error

	// InWorkConn accept work connections registered to server.
	InWorkConn(conn frpNet.Conn)
	Close()
	log.Logger
}

func NewProxy(ctl *Control, pxyConf config.ProxyConf) (pxy Proxy) {
	baseProxy := BaseProxy{
		ctl:    ctl,
		Logger: log.NewPrefixLogger(pxyConf.GetName()),
	}
	switch cfg := pxyConf.(type) {
	case *config.TcpProxyConf:
		pxy = &TcpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case *config.UdpProxyConf:
		pxy = &UdpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case *config.FtpProxyConf:
		pxy = &FtpProxy{
			BaseProxy:	baseProxy,
			cfg:		cfg,
		}
	case *config.HttpProxyConf:
		pxy = &HttpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case *config.HttpsProxyConf:
		pxy = &HttpsProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	}
	return
}

type BaseProxy struct {
	ctl    *Control
	closed bool
	mu     sync.RWMutex
	log.Logger
}

// TCP
type TcpProxy struct {
	BaseProxy

	cfg         *config.TcpProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *TcpProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *TcpProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *TcpProxy) InWorkConn(conn frpNet.Conn) {
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, pxy.proxyPlugin, &pxy.cfg.BaseProxyConf, conn)
}

// ftp
type FtpProxy struct {
	BaseProxy
	
	cfg			*config.FtpProxyConf
}

func (pxy *FtpProxy) Run() (err error) {
	return
}

func (pxy *FtpProxy) Close() {
}

func (pxy *FtpProxy) InWorkConn(conn frpNet.Conn) {
	HandleFtpControlConnection(&pxy.cfg.LocalSvrConf, &pxy.BaseProxy,  pxy.cfg, conn)
}

func HandleFtpControlConnection(localInfo *config.LocalSvrConf, bp *BaseProxy, cfg	*config.FtpProxyConf, workConn frpNet.Conn) {
	ftpConn, err := frpNet.ConnectTcpServer(fmt.Sprintf("%s:%d", localInfo.LocalIp, localInfo.LocalPort))
	if err != nil {
		workConn.Error("connect to local service [%s:%d] error: %v", localInfo.LocalIp, localInfo.LocalPort, err)
		return
	}
	
	go JoinFtpControl(ftpConn, workConn, bp, cfg, localInfo)
}

// todo
func SetFtpDataProxyLocalServer(bp *BaseProxy, cfg *config.FtpProxyConf, localInfo *config.LocalSvrConf, port int) (err error) {
	var (
		name 	string
		msg		msg.NewProxy
	)
	cfg.UnMarshalToMsg(&msg)
	name = fmt.Sprintf("%s%d", msg.ProxyName, msg.RemoteDataPort)
	ftpDataConf, ok := bp.ctl.getProxyConfByName(name)
	if !ok {
		fmt.Printf("get ftpDataConf failed by %s\n", name)
		err = fmt.Errorf("get ftpDataConf failed by %s\n", name)
		return
	}
	ftpDataConf.FillLocalServer(localInfo.LocalIp, port)
	return
}

// handler for ftp work connection
func JoinFtpControl(fc io.ReadWriteCloser, fs io.ReadWriteCloser, bp *BaseProxy, cfg *config.FtpProxyConf, localInfo *config.LocalSvrConf) (inCount int64, outCount int64) {
	var wait sync.WaitGroup
	ftpPipe := func(to io.ReadWriteCloser, from io.ReadWriteCloser, count *int64) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()
		
		for {
			data := make([]byte, 1024)
			n, err := from.Read(data)
			if n <= 0 || err != nil {
				fmt.Printf("from.Read failed, n is %d, err is %v\n", n, err)
				return
			}
	
			msg := string(data[:n])	
			code, _ := strconv.Atoi(msg[:3])
			if code == 227 {
				port:= GetFtpPasvPort(msg)
				if port != 0 {
					// create data session
					newPort := int(cfg.RemoteDataPort)
					SetFtpDataProxyLocalServer(bp, cfg, localInfo, newPort)
					newMsg := NewFtpPasv(newPort)
					fmt.Printf("msg: [%s] newMsg: [%s]", msg, newMsg)
					to.Write([]byte(newMsg))
					break
				} else {
					to.Write(data[:n])
				}
			} else if code == 211 {
				to.Write(data[:n])
				if n < 87 {
					n, err = from.Read(data)
					to.Write(data[:n])
				}
			} else {
				to.Write(data[:n])
			}
			
			n, err = to.Read(data)
			if n <= 0 || err != nil {
				fmt.Printf("to.Read failed, n is %d, err is %v\n", n, err)
				return
			}
			from.Write(data[:n])
		}
		
		*count, _ = io.Copy(to, from)
	}
	
	wait.Add(2)
	go ftpPipe(fs, fc, &inCount)
	wait.Wait()
	return
}

// HTTP
type HttpProxy struct {
	BaseProxy

	cfg         *config.HttpProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *HttpProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *HttpProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *HttpProxy) InWorkConn(conn frpNet.Conn) {
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, pxy.proxyPlugin, &pxy.cfg.BaseProxyConf, conn)
}

// HTTPS
type HttpsProxy struct {
	BaseProxy

	cfg         *config.HttpsProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *HttpsProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *HttpsProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *HttpsProxy) InWorkConn(conn frpNet.Conn) {
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, pxy.proxyPlugin, &pxy.cfg.BaseProxyConf, conn)
}

// UDP
type UdpProxy struct {
	BaseProxy

	cfg *config.UdpProxyConf

	localAddr *net.UDPAddr
	readCh    chan *msg.UdpPacket

	// include msg.UdpPacket and msg.Ping
	sendCh   chan msg.Message
	workConn frpNet.Conn
}

func (pxy *UdpProxy) Run() (err error) {
	pxy.localAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", pxy.cfg.LocalIp, pxy.cfg.LocalPort))
	if err != nil {
		return
	}
	return
}

func (pxy *UdpProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()

	if !pxy.closed {
		pxy.closed = true
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		if pxy.readCh != nil {
			close(pxy.readCh)
		}
		if pxy.sendCh != nil {
			close(pxy.sendCh)
		}
	}
}

func (pxy *UdpProxy) InWorkConn(conn frpNet.Conn) {
	pxy.Info("incoming a new work connection for udp proxy, %s", conn.RemoteAddr().String())
	// close resources releated with old workConn
	pxy.Close()

	pxy.mu.Lock()
	pxy.workConn = conn
	pxy.readCh = make(chan *msg.UdpPacket, 1024)
	pxy.sendCh = make(chan msg.Message, 1024)
	pxy.closed = false
	pxy.mu.Unlock()

	workConnReaderFn := func(conn net.Conn, readCh chan *msg.UdpPacket) {
		for {
			var udpMsg msg.UdpPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				pxy.Warn("read from workConn for udp error: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				pxy.Trace("get udp package from workConn: %s", udpMsg.Content)
				readCh <- &udpMsg
			}); errRet != nil {
				pxy.Info("reader goroutine for udp work connection closed: %v", errRet)
				return
			}
		}
	}
	workConnSenderFn := func(conn net.Conn, sendCh chan msg.Message) {
		defer func() {
			pxy.Info("writer goroutine for udp work connection closed")
		}()
		var errRet error
		for rawMsg := range sendCh {
			switch m := rawMsg.(type) {
			case *msg.UdpPacket:
				pxy.Trace("send udp package to workConn: %s", m.Content)
			case *msg.Ping:
				pxy.Trace("send ping message to udp workConn")
			}
			if errRet = msg.WriteMsg(conn, rawMsg); errRet != nil {
				pxy.Error("udp work write error: %v", errRet)
				return
			}
		}
	}
	heartbeatFn := func(conn net.Conn, sendCh chan msg.Message) {
		var errRet error
		for {
			time.Sleep(time.Duration(30) * time.Second)
			if errRet = errors.PanicToError(func() {
				sendCh <- &msg.Ping{}
			}); errRet != nil {
				pxy.Trace("heartbeat goroutine for udp work connection closed")
				break
			}
		}
	}

	go workConnSenderFn(pxy.workConn, pxy.sendCh)
	go workConnReaderFn(pxy.workConn, pxy.readCh)
	go heartbeatFn(pxy.workConn, pxy.sendCh)
	udp.Forwarder(pxy.localAddr, pxy.readCh, pxy.sendCh)
}

func NewFtpPasv(port int) (newMsg string) {
	p1 := port / 256
	p2 := port - (p1 * 256)
	
	quads := strings.Split(config.ClientCommonCfg.ServerAddr, ".")
	newMsg = fmt.Sprintf("227 Entering Passive Mode (%s,%s,%s,%s,%d,%d).", quads[0], quads[1], quads[2], quads[3], p1, p2)
	return
}

func GetFtpPasvPort(msg string) (port int) {
	port = 0
	if len(msg) < 45 {
		return 
	}
	
	start := strings.Index(msg, "(")
	end := strings.LastIndex(msg, ")")
	if start == -1 || end == -1 {
		return
	}
	
	// We have to split the response string
	pasvData := strings.Split(msg[start+1:end], ",")

	if len(pasvData) < 6 {
		return
	}

	// Let's compute the port number
	portPart1, err1 := strconv.Atoi(pasvData[4])
	if err1 != nil {
		return
	}

	portPart2, err2 := strconv.Atoi(pasvData[5])
	if err2 != nil {
		return
	}

	// Recompose port
	port = portPart1*256 + portPart2
	return
}

func CreateFtpDataProxy(bp *BaseProxy, port int, name string) {
	cfg := config.NewConfByType(consts.FtpProxy)
	
	
	newName := fmt.Sprintf("%s%d", name, port)
	var newProxyMsg msg.NewProxy
	newProxyMsg.RemotePort = int64(port)
	newProxyMsg.ProxyName =  newName
	newProxyMsg.ProxyType = consts.FtpProxy
	newProxyMsg.UseEncryption = false
	newProxyMsg.UseCompression = false
	
	bp.ctl.sendCh <- &newProxyMsg
	
	cfg.LoadFromMsg(&newProxyMsg)
	bp.ctl.pxyCfgs[newName] = cfg
}


// Common handler for tcp work connections.
func HandleTcpWorkConnection(localInfo *config.LocalSvrConf, proxyPlugin plugin.Plugin,
							baseInfo *config.BaseProxyConf, workConn frpNet.Conn) {

	var (
		remote io.ReadWriteCloser
		err    error
	)
	remote = workConn
	if baseInfo.UseEncryption {
		remote, err = tcp.WithEncryption(remote, []byte(config.ClientCommonCfg.PrivilegeToken))
		if err != nil {
			workConn.Error("create encryption stream error: %v", err)
			return
		}
	}
	if baseInfo.UseCompression {
		remote = tcp.WithCompression(remote)
	}

	if proxyPlugin != nil {
		// if plugin is set, let plugin handle connections first
		workConn.Debug("handle by plugin: %s", proxyPlugin.Name())
		proxyPlugin.Handle(remote)
		workConn.Debug("handle by plugin finished")
		return
	} else {
		localConn, err := frpNet.ConnectTcpServer(fmt.Sprintf("%s:%d", localInfo.LocalIp, localInfo.LocalPort))
		if err != nil {
			workConn.Error("connect to local service [%s:%d] error: %v", localInfo.LocalIp, localInfo.LocalPort, err)
			return
		}

		workConn.Debug("join connections, localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
			localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())
		tcp.Join(localConn, remote)
		workConn.Debug("join connections closed")
	}
}
