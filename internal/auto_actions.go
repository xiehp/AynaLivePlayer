package internal

import (
	"AynaLivePlayer/core/events"
	"AynaLivePlayer/global"
	"AynaLivePlayer/gui/gctx"
	"AynaLivePlayer/internal/liveroom"
	"AynaLivePlayer/internal/player"
	"time"
)

// InitAutoActions 初始化所有自动执行的小功能。
// 这是定制功能的统一入口，未来新增小功能只需在此函数内添加调用，
// 无需再修改 internal.go 等原作者文件。
func InitAutoActions() {
	player.InitPlayStatePersistence()
	autoOpenPlayerWindow()
	autoReconnectRooms()
}

// SaveAutoActions 程序退出前保存所有定制功能的状态。
func SaveAutoActions() {
	player.SavePlayState()
}

// autoOpenPlayerWindow 初始化完成后延迟 10 秒自动打开底部「播放器」小窗。
func autoOpenPlayerWindow() {
	log := global.Logger.WithPrefix("AutoActions")
	log.Info("[AutoPlayerWindow] scheduling auto-open 10s after init")
	time.AfterFunc(10*time.Second, func() {
		log.Info("[AutoPlayerWindow] opening player window via GUISetPlayerWindowOpenCmd")
		_ = global.EventBus.PublishToChannel(
			gctx.EventChannel,
			events.GUISetPlayerWindowOpenCmd,
			events.GUISetPlayerWindowOpenCmdEvent{SetOpen: true},
		)
	})
}

// autoReconnectRooms 每 5 分钟对勾选自动连接且已断开的直播间执行重连。
func autoReconnectRooms() {
	log := global.Logger.WithPrefix("AutoActions")
	log.Info("[AutoReconnect] starting reconnect timer (interval: 5min)")
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			liveroom.ReconnectAutoRooms()
		}
	}()
}
