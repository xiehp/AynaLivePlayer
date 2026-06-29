package player

import (
	"AynaLivePlayer/core/events"
	"AynaLivePlayer/core/model"
	"AynaLivePlayer/global"
	"AynaLivePlayer/internal/playlist"
	"AynaLivePlayer/pkg/eventbus"
	"encoding/json"
	"os"
	"sync"
	"time"
)

const playStatePath = "./config/play_state.json"

// PlayState 播放状态持久化结构
type PlayState struct {
	CurrentSong     model.Media   `json:"current_song"`
	PendingPlaylist []model.Media `json:"pending_playlist"`
	PlaylistIndex   int           `json:"playlist_index"`
	SavedAt         time.Time     `json:"saved_at"`
}

var (
	stateFileMutex  sync.Mutex
	currentSong     model.Media
	currentSongLock sync.RWMutex
)

// InitPlayStatePersistence 启动时检查是否有上次的播放状态，有则恢复。
// 先等待 10 秒初始延迟，然后订阅 PlayerPlayingUpdate 等待播放器就绪（Removed=false），
// 30 秒超时后放弃等待。恢复时将保存的当前歌曲 + 待播列表一次性插入 PlayerPlaylist，
// 设置 Index=0 并通过 PlayerPlayNextCmd 从当前歌曲开始续播。
func InitPlayStatePersistence() {
	log := global.Logger.WithPrefix("PlayState")
	state := readPlayState()
	if state == nil {
		return
	}

	// 问题2：订阅 PlayerPlayCmd 追踪当前播放歌曲，保存时使用
	_ = global.EventBus.Subscribe("", events.PlayerPlayCmd, "player.state_persist.track_current",
		func(evnt *eventbus.Event) {
			media := evnt.Data.(events.PlayerPlayCmdEvent).Media
			currentSongLock.Lock()
			currentSong = media
			currentSongLock.Unlock()
		})

	log.Info("[Restore] scheduling delayed restore (initial delay: 10s)")
	time.AfterFunc(10*time.Second, func() {
		log.Info("[Restore] waiting for player to become ready...")

		// 问题1：等待 PlayerPlayingUpdate 中 Removed=false 信号
		readyCh := make(chan struct{}, 1)
		handlerName := "player.state_persist.restore_ready"

		_ = global.EventBus.Subscribe("", events.PlayerPlayingUpdate, handlerName,
			func(evnt *eventbus.Event) {
				data := evnt.Data.(events.PlayerPlayingUpdateEvent)
				if !data.Removed {
					select {
					case readyCh <- struct{}{}:
					default:
					}
				}
			})

		// 等待播放器就绪信号或 30 秒超时
		select {
		case <-readyCh:
			log.Info("[Restore] player is ready, proceeding with restore")
		case <-time.After(30 * time.Second):
			log.Info("[Restore] timeout waiting for player ready (30s), proceeding anyway")
		}

		_ = global.EventBus.Unsubscribe(events.PlayerPlayingUpdate, handlerName)

		// 问题2：合并当前歌曲 + 待播列表，一次性插入 PlayerPlaylist
		all := make([]model.Media, 0)
		if state.CurrentSong.Info.Title != "" {
			log.Infof("[Restore] prepending current song: %s", state.CurrentSong.Info.Title)
			all = append(all, state.CurrentSong)
		}
		if len(state.PendingPlaylist) > 0 {
			log.Infof("[Restore] appending %d pending songs", len(state.PendingPlaylist))
			for _, m := range state.PendingPlaylist {
				log.Infof("[Restore]   %s", m.Info.Title)
			}
			all = append(all, state.PendingPlaylist...)
		}

		if len(all) > 0 {
			for _, m := range all {
				playlist.PlayerPlaylist.Insert(-1, m)
			}
			playlist.PlayerPlaylist.Index = 0

			log.Info("[Restore] publishing PlayerPlayNextCmd to start playback from current song")
			_ = global.EventBus.Publish(events.PlayerPlayNextCmd, events.PlayerPlayNextCmdEvent{})
		}

		// 恢复后清理状态文件
		stateFileMutex.Lock()
		_ = os.Remove(playStatePath)
		stateFileMutex.Unlock()
		log.Info("[Restore] state file removed after successful restore")
	})
}

// SavePlayState 关闭时保存当前播放状态，供下次启动恢复。
// 保存当前播放歌曲 + 待播列表。
func SavePlayState() {
	stateFileMutex.Lock()
	defer stateFileMutex.Unlock()

	log := global.Logger.WithPrefix("PlayState")
	log.Info("[Save] saving play state on shutdown")

	// 获取当前播放歌曲
	currentSongLock.RLock()
	cs := currentSong
	currentSongLock.RUnlock()

	// 收集待播列表
	var pending []model.Media
	var plIndex int
	if playlist.PlayerPlaylist != nil {
		pending = playlist.PlayerPlaylist.CopyMedia()
		plIndex = playlist.PlayerPlaylist.Index
	}

	state := PlayState{
		CurrentSong:     cs,
		PendingPlaylist: pending,
		PlaylistIndex:   plIndex,
		SavedAt:         time.Now(),
	}

	log.Infof("[Save] current_song=%q, pending_songs=%d, playlist_index=%d",
		cs.Info.Title, len(pending), plIndex)
	for i, m := range pending {
		log.Infof("[Save]   pending[%d]: %s", i, m.Info.Title)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Errorf("[Save] marshal failed: %v", err)
		return
	}

	if err := os.WriteFile(playStatePath, data, 0644); err != nil {
		log.Errorf("[Save] write to %s failed: %v", playStatePath, err)
		return
	}
	log.Infof("[Save] written to %s (%d bytes)", playStatePath, len(data))
}

func readPlayState() *PlayState {
	log := global.Logger.WithPrefix("PlayState")
	data, err := os.ReadFile(playStatePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("[Restore] failed to read state file %s: %v", playStatePath, err)
		}
		return nil
	}

	log.Infof("[Restore] found state file: %s (%d bytes)", playStatePath, len(data))

	var state PlayState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Errorf("[Restore] failed to parse state file: %v", err)
		return nil
	}

	log.Infof("[Restore] saved state: current_song=%q, pending=%d songs, playlist_index=%d, saved_at=%s",
		state.CurrentSong.Info.Title, len(state.PendingPlaylist), state.PlaylistIndex,
		state.SavedAt.Format("2006-01-02 15:04:05"))
	return &state
}
