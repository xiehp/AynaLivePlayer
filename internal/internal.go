package internal

import (
	"AynaLivePlayer/global"
	"AynaLivePlayer/internal/controller"
	"AynaLivePlayer/internal/liveroom"
	"AynaLivePlayer/internal/player"
	"AynaLivePlayer/internal/playlist"
	"AynaLivePlayer/internal/plugins"
	"AynaLivePlayer/internal/source"
	"AynaLivePlayer/internal/sysmediacontrol"
	"AynaLivePlayer/internal/updater"
	"AynaLivePlayer/pkg/config"
	"AynaLivePlayer/plugin/diange"
	"AynaLivePlayer/plugin/durationmgmt"
	"AynaLivePlayer/plugin/qiege"
	"AynaLivePlayer/plugin/sourcelogin"
	"AynaLivePlayer/plugin/textinfo"
	"AynaLivePlayer/plugin/wshub"
	"AynaLivePlayer/plugin/yinliang"
	"time"
)

func Initialize() {
	startTime := time.Now()
	global.Logger.Infof("[InitTimeline] Initialize() started at %s", startTime.Format("15:04:05.000"))

	player.SetupPlayer()
	global.Logger.Infof("[InitTimeline] Step 1: SetupPlayer done (+%v)", time.Since(startTime))

	source.Initialize()
	global.Logger.Infof("[InitTimeline] Step 2: source.Initialize done (+%v)", time.Since(startTime))

	playlist.Initialize()
	global.Logger.Infof("[InitTimeline] Step 3: playlist.Initialize done (+%v)", time.Since(startTime))

	controller.Initialize()
	global.Logger.Infof("[InitTimeline] Step 4: controller.Initialize done (+%v)", time.Since(startTime))

	player.InitPlayStatePersistence()
	global.Logger.Infof("[InitTimeline] Step 5: InitPlayStatePersistence done (+%v)", time.Since(startTime))

	liveroom.Initialize()
	global.Logger.Infof("[InitTimeline] Step 6: liveroom.Initialize done (+%v)", time.Since(startTime))

	plugins.Initialize()
	plugins.LoadPlugins(
		diange.NewDiange(), qiege.NewQiege(), yinliang.NewYinliang(), sourcelogin.NewSourceLogin(),
		textinfo.NewTextInfo(),
		durationmgmt.NewMaxDuration(),
		wshub.NewWsHub(),
	)
	global.Logger.Infof("[InitTimeline] Step 7: plugins loaded done (+%v)", time.Since(startTime))

	updater.Initialize()
	global.Logger.Infof("[InitTimeline] Step 8: updater.Initialize done (+%v)", time.Since(startTime))

	if config.General.EnableSMC {
		sysmediacontrol.InitSystemMediaControl()
		global.Logger.Infof("[InitTimeline] Step 9: sysmediacontrol.Init done (+%v)", time.Since(startTime))
	}

	global.Logger.Infof("[InitTimeline] Initialize() completed, total=%v", time.Since(startTime))
}

func Stop() {
	if config.General.EnableSMC {
		sysmediacontrol.Destroy()
	}
	liveroom.StopAndSave()
	playlist.Close()
	plugins.ClosePlugins()
	player.SavePlayState()
	player.StopPlayer()
}
