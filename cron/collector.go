package cron

import (
	"time"

	"github.com/open-falcon/agent/funcs"
	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"
)

// 初始化历史数据,只有cpu和disk需要历史数据
func InitDataHistory() {
	for {
		funcs.UpdateCpuStat()
		funcs.UpdateDiskStats()
		time.Sleep(g.COLLECT_INTERVAL)
	}
}

func Collect() {
	// 配置信息判断
	if !g.Config().Transfer.Enabled {
		return
	}

	if len(g.Config().Transfer.Addrs) == 0 {
		return
	}
    // 读取mapper中的FuncsAndInterval集,并通过不同的goroutine运行
	for _, v := range funcs.Mappers {
		go collect(int64(v.Interval), v.Fs)
	}
}

// 间隔采集信息
func collect(sec int64, fns []func() []*model.MetricValue) {
	// 启动断续器,间隔执行
	t := time.NewTicker(time.Second * time.Duration(sec)).C
	for {
		<-t

		hostname, err := g.Hostname()
		if err != nil {
			continue
		}

		mvs := []*model.MetricValue{}
		// 读取忽略metric名单
			ignoreMetrics := g.Config().IgnoreMetrics
		// 从funcs的list中取出每个采集函数
		for _, fn := range fns {
			// 执行采集函数
			items := fn()
			if items == nil {
				continue
			}

			if len(items) == 0 {
				continue
			}
			// 读取采集数据,根据忽略的metric忽略部分采集数据
			for _, mv := range items {
				if b, ok := ignoreMetrics[mv.Metric]; ok && b {
					continue
				} else {
					mvs = append(mvs, mv)
				}
			}
		}
		// 获取上报时间
		now := time.Now().Unix()
		// 设置上报采集项的间隔、agent主机、上报时间
		for j := 0; j < len(mvs); j++ {
			mvs[j].Step = sec
			mvs[j].Endpoint = hostname
			mvs[j].Timestamp = now
		}
		// 调用transfer发送采集数据
		g.SendToTransfer(mvs)

	}
}
