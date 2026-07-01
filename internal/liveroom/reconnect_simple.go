package liveroom

import (
	"AynaLivePlayer/global"
	"AynaLivePlayer/pkg/logger"
	liveroomsdk "github.com/AynaLivePlayer/liveroom-sdk"
	"sync"
	"time"
)

var (
	reconLogOnce sync.Once
	reconLog     logger.ILogger
)

func getReconLog() logger.ILogger {
	reconLogOnce.Do(func() {
		reconLog = global.Logger.WithPrefix("Reconnect")
	})
	return reconLog
}

func reconnectAfterDelay(room liveroomsdk.ILiveRoom) error {
	log := getReconLog()
	log.Infof("disconnecting %s", room.Config().Identifier())
	room.Disconnect()
	log.Info("waiting 5s before connect")
	time.Sleep(5 * time.Second)
	log.Infof("connecting %s", room.Config().Identifier())
	err := room.Connect()
	if err != nil {
		log.Errorf("connect %s failed: %s", room.Config().Identifier(), err)
	} else {
		log.Infof("connect %s succeeded", room.Config().Identifier())
	}
	return err
}

// ReconnectAutoRooms 遍历所有勾选了自动连接的房间，对已断开的执行重连。
func ReconnectAutoRooms() {
	log := getReconLog()
	for _, r := range liveRooms {
		if r.model.Config.AutoConnect && !r.model.Status {
			log.Infof("auto reconnect: %s", r.room.Config().Identifier())
			if err := reconnectAfterDelay(r.room); err != nil {
				log.Errorf("auto reconnect %s failed: %s", r.room.Config().Identifier(), err)
			}
		}
	}
}
