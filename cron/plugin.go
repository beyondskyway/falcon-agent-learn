package cron

import (
	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/agent/plugins"
	"github.com/open-falcon/common/model"
	"log"
	"strings"
	"time"
)

func SyncMinePlugins() {
	// 判断配置
	if !g.Config().Plugin.Enabled {
		return
	}

	if !g.Config().Heartbeat.Enabled {
		return
	}

	if g.Config().Heartbeat.Addr == "" {
		return
	}

	go syncMinePlugins()
}

func syncMinePlugins() {

	var (
		timestamp  int64 = -1
		pluginDirs []string
	)

	duration := time.Duration(g.Config().Heartbeat.Interval) * time.Second

	for {
		time.Sleep(duration)

		hostname, err := g.Hostname()
		if err != nil {
			continue
		}

		req := model.AgentHeartbeatRequest{
			Hostname: hostname,
		}

		var resp model.AgentPluginsResponse
		// 调用rpc提供的hbs接口,返回plugin
		err = g.HbsClient.Call("Agent.MinePlugins", req, &resp)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}
		// 保证时间顺序正确
		if resp.Timestamp <= timestamp {
			continue
		}

		pluginDirs = resp.Plugins
		// 存放时间保证最新
		timestamp = resp.Timestamp

		if g.Config().Debug {
			log.Println(&resp)
		}
		// 无插件则清空plugin
		if len(pluginDirs) == 0 {
			plugins.ClearAllPlugins()
		}

		desiredAll := make(map[string]*plugins.Plugin)
		// 读取所有plugin
		for _, p := range pluginDirs {
			// 根据相对路径生成plugin的map
			underOneDir := plugins.ListPlugins(strings.Trim(p, "/"))
			// 为什么不直接赋给desiredAll  ????
			for k, v := range underOneDir {
				desiredAll[k] = v
			}
		}
		// 停止不需要的插件,启动增加的插件
		plugins.DelNoUsePlugins(desiredAll)
		plugins.AddNewPlugins(desiredAll)

	}
}
