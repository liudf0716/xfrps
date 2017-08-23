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
	"sync"
	"time"

	"github.com/KunTengRom/xfrps/models/config"
	"github.com/KunTengRom/xfrps/utils/log"
	"github.com/KunTengRom/xfrps/utils/metric"
)

const (
	ReserveDays = 7
)

var globalStats *ServerStatistics

type ServerStatistics struct {
	TotalTrafficIn  metric.DateCounter
	TotalTrafficOut metric.DateCounter
	CurConns        metric.Counter

	// counter for clients
	ClientCounts metric.Counter

	// counter for proxy types
	ProxyTypeCounts map[string]metric.Counter

	// statistics for different proxies
	// key is proxy name
	ProxyStatistics map[string]*ProxyStatistics

	// statistics for client
	// key is runid
	ClientStatistics map[string]*ClientStatistics

	mu sync.Mutex
}

type ClientStatistics struct {
	Online        int
	ProxyNum      metric.Counter
	ConnNum       metric.Counter
	LastStartTime time.Time
	LastCloseTime time.Time
}

type ProxyStatistics struct {
	Name          string
	RunId         string
	ProxyType     string
	TrafficIn     metric.DateCounter
	TrafficOut    metric.DateCounter
	CurConns      metric.Counter
	LastStartTime time.Time
	LastCloseTime time.Time
}

func init() {
	globalStats = &ServerStatistics{
		TotalTrafficIn:  metric.NewDateCounter(ReserveDays),
		TotalTrafficOut: metric.NewDateCounter(ReserveDays),
		CurConns:        metric.NewCounter(),

		ClientCounts:    metric.NewCounter(),
		ProxyTypeCounts: make(map[string]metric.Counter),

		ProxyStatistics: make(map[string]*ProxyStatistics),

		ClientStatistics: make(map[string]*ClientStatistics),
	}

	go func() {
		for {
			time.Sleep(12 * time.Hour)
			log.Debug("start to clear useless proxy statistics data...")
			StatsClearUselessInfo()
			log.Debug("finish to clear useless proxy statistics data")
		}
	}()
}

func StatsClearUselessInfo() {
	// To check if there are proxies that closed than 7 days and drop them.
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	for name, data := range globalStats.ProxyStatistics {
		if !data.LastCloseTime.IsZero() && time.Since(data.LastCloseTime) > time.Duration(7*24)*time.Hour {
			delete(globalStats.ProxyStatistics, name)
			log.Trace("clear proxy [%s]'s statistics data, lastCloseTime: [%s]", name, data.LastCloseTime.String())
		}
	}

	for runid, data := range globalStats.ClientStatistics {
		if !data.LastCloseTime.IsZero() && time.Since(data.LastCloseTime) > time.Duration(7*24)*time.Hour {
			delete(globalStats.ClientStatistics, runid)
			log.Trace("clear client [%s]'s statistics data, lastCloseTime: [%s]", runid, data.LastCloseTime.String())
		}
	}
}

func StatsNewClient(runid string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.ClientCounts.Inc(1)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		clientStats, ok := globalStats.ClientStatistics[runid]
		if !ok {
			clientStats = &ClientStatistics{
				ProxyNum: metric.NewCounter(),
				ConnNum:  metric.NewCounter(),
			}
			globalStats.ClientStatistics[runid] = clientStats
		}
		clientStats.LastStartTime = time.Now()
		clientStats.Online = 1
	}
}

func StatsCloseClient(runid string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.ClientCounts.Dec(1)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		clientStats, ok := globalStats.ClientStatistics[runid]
		if ok {
			clientStats.LastCloseTime = time.Now()
			clientStats.Online = 0
		}

	}
}

func StatsNewProxy(name string, proxyType string, runid string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		counter, ok := globalStats.ProxyTypeCounts[proxyType]
		if !ok {
			counter = metric.NewCounter()
		}
		counter.Inc(1)
		globalStats.ProxyTypeCounts[proxyType] = counter

		if clientStats, ok := globalStats.ClientStatistics[runid]; ok {
			clientStats.ProxyNum.Inc(1)
		}

		proxyStats, ok := globalStats.ProxyStatistics[name]
		if !(ok && proxyStats.ProxyType == proxyType) {
			proxyStats = &ProxyStatistics{
				Name:       name,
				RunId:      runid,
				ProxyType:  proxyType,
				CurConns:   metric.NewCounter(),
				TrafficIn:  metric.NewDateCounter(ReserveDays),
				TrafficOut: metric.NewDateCounter(ReserveDays),
			}
			globalStats.ProxyStatistics[name] = proxyStats
		}
		proxyStats.LastStartTime = time.Now()
	}
}

func StatsCloseProxy(proxyName string, proxyType string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		if counter, ok := globalStats.ProxyTypeCounts[proxyType]; ok {
			counter.Dec(1)
		}
		if proxyStats, ok := globalStats.ProxyStatistics[proxyName]; ok {
			proxyStats.LastCloseTime = time.Now()
			if clientStats, ok := globalStats.ClientStatistics[proxyStats.RunId]; ok {
				clientStats.ProxyNum.Dec(1)
			}
		}
	}
}

func StatsOpenConnection(name string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.CurConns.Inc(1)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.CurConns.Inc(1)
			globalStats.ProxyStatistics[name] = proxyStats
			if clientStats, ok := globalStats.ClientStatistics[proxyStats.RunId]; ok {
				clientStats.ConnNum.Inc(1)
			}
		}
	}
}

func StatsCloseConnection(name string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.CurConns.Dec(1)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.CurConns.Dec(1)
			globalStats.ProxyStatistics[name] = proxyStats
			if clientStats, ok := globalStats.ClientStatistics[proxyStats.RunId]; ok {
				clientStats.ConnNum.Dec(1)
			}
		}
	}
}

func StatsAddTrafficIn(name string, trafficIn int64) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.TotalTrafficIn.Inc(trafficIn)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()

		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.TrafficIn.Inc(trafficIn)
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

func StatsAddTrafficOut(name string, trafficOut int64) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.TotalTrafficOut.Inc(trafficOut)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()

		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.TrafficOut.Inc(trafficOut)
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

// Functions for getting server stats.
type ServerStats struct {
	TotalTrafficIn  int64
	TotalTrafficOut int64
	CurConns        int64
	ClientCounts    int64
	ProxyTypeCounts map[string]int64
}

func StatsGetServer() *ServerStats {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	s := &ServerStats{
		TotalTrafficIn:  globalStats.TotalTrafficIn.TodayCount(),
		TotalTrafficOut: globalStats.TotalTrafficOut.TodayCount(),
		CurConns:        globalStats.CurConns.Count(),
		ClientCounts:    globalStats.ClientCounts.Count(),
		ProxyTypeCounts: make(map[string]int64),
	}
	for k, v := range globalStats.ProxyTypeCounts {
		s.ProxyTypeCounts[k] = v.Count()
	}
	return s
}

type ClientStats struct {
	RunId         string
	ProxyNum      metric.Counter
	ConnNum       metric.Counter
	LastStartTime string
	LastCloseTime string
}

// online is 1, means online; otherwise offline
func StatsGetClient(online int) []*ClientStats {
	res := make([]*ClientStats, 0)
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	for runid, clientStats := range globalStats.ClientStatistics {
		if clientStats.Online != online {
			continue
		}

		ps := &ClientStats{
			RunId:    runid,
			ProxyNum: clientStats.ProxyNum,
			ConnNum:  clientStats.ConnNum,
		}
		if !clientStats.LastStartTime.IsZero() {
			ps.LastStartTime = clientStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !clientStats.LastCloseTime.IsZero() {
			ps.LastCloseTime = clientStats.LastCloseTime.Format("01-02 15:04:05")
		}
		res = append(res, ps)
	}
	return res
}

type ProxyStats struct {
	Name            string
	RunId           string
	Type            string
	TodayTrafficIn  int64
	TodayTrafficOut int64
	LastStartTime   string
	LastCloseTime   string
	CurConns        int64
}

func StatsGetProxiesByType(proxyType string) []*ProxyStats {
	res := make([]*ProxyStats, 0)
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	for name, proxyStats := range globalStats.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		ps := &ProxyStats{
			Name:            name,
			RunId:           proxyStats.RunId,
			Type:            proxyStats.ProxyType,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        proxyStats.CurConns.Count(),
		}
		if !proxyStats.LastStartTime.IsZero() {
			ps.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			ps.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		res = append(res, ps)
	}
	return res
}

type ProxyTrafficInfo struct {
	Name       string
	TrafficIn  []int64
	TrafficOut []int64
}

func StatsGetProxyTraffic(name string) (res *ProxyTrafficInfo) {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	proxyStats, ok := globalStats.ProxyStatistics[name]
	if ok {
		res = &ProxyTrafficInfo{
			Name: name,
		}
		res.TrafficIn = proxyStats.TrafficIn.GetLastDaysCount(ReserveDays)
		res.TrafficOut = proxyStats.TrafficOut.GetLastDaysCount(ReserveDays)
	}
	return
}
