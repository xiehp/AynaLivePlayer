# 变更记录模板

这份文件用于记录每一次具体修改。以后每做完一件事，尽量在这里补一条简短记录。

## 记录格式

- 日期：
- 目标：
- 变更内容：
- 涉及文件：
- 影响范围：
- 是否需要重放补丁：
- 回滚方式：

## 示例

- 日期：2026-06-20
- 目标：补充最小改动原则
- 变更内容：要求原文件只保留挂载点，新增逻辑全部放到新文件
- 涉及文件：docs/development-norms.md
- 影响范围：后续所有定制开发
- 是否需要重放补丁：是
- 回滚方式：删除对应文档段落并恢复旧版本

## 2026-06-26：直播间断线重连

- 日期：2026-06-26
- 目标：实现弹幕房间断线后自动重连
- 变更内容：
  - 新增 ReconnectScheduler 调度器，管理断线检测、重试计数、间隔等待、放弃通知
  - 规则：最多重试 3 次，每次间隔 2 分钟；3 次全部失败后发布 LiveRoomReconnectGiveUp 事件
  - 用户手动断开时自动停止调度器，避免误触发重连
  - 关键节点（断线检测、每次重试开始/成功/失败/放弃）写入详细日志
- 涉及文件：
  - internal/liveroom/reconnect.go（新增，重连调度器核心逻辑）
  - internal/liveroom/liveroom.go（修改：liveroom 结构体增加 scheduler 字段，OnStatusChange 回调委托调度器，disconnect 操作前停止调度器，shutdown 时停止所有调度器）
  - core/events/liveroom.go（新增 LiveRoomReconnectGiveUp 事件及数据体）
- 影响范围：直播间断线后自动重连行为；其他模块可订阅 LiveRoomReconnectGiveUp 事件感知重连最终失败
- 是否需要重放补丁：是。原作者更新后需重新应用以下入口改动：
  - liveroom.go 中 liveroom 结构体的 scheduler 字段
  - liveroom.go 中 OnStatusChange 回调内的 scheduler.OnStatusChange 调用
  - liveroom.go 中三处 scheduler.Stop() 调用
  - reconnect.go 完整放回 internal/liveroom/ 目录
  - events/liveroom.go 中新增事件常量与结构体
- 回滚方式：删除 reconnect.go，还原 liveroom.go 与 events/liveroom.go 中被修改的段落

## 2026-06-26：播放状态持久化

- 日期：2026-06-26
- 目标：实现播放状态持久化，重启软件后自动恢复当前歌曲和待播列表
- 变更内容：
  - 新增 PlayState JSON 结构，包含 current_song、pending_playlist、playlist_index、saved_at 四个字段
  - 状态文件路径为 ./config/play_state.json
  - 播放器每次切歌时自动保存（订阅 PlayerPlayCmd 事件）
  - 待播列表每次增删时自动保存（订阅 PlaylistDetailUpdate 事件）
  - 程序正常关闭时通过 SavePlayState() 保存最终状态
  - 程序启动时 InitPlayStatePersistence() 检查状态文件：
    - 同步恢复待播列表到 PlayerPlaylist（不依赖事件总线）
    - 事件总线启动后自动恢复当前歌曲并删除状态文件
- 涉及文件：
  - internal/player/state_persist.go（新增，持久化核心逻辑：初始化、保存、恢复、清理）
  - internal/internal.go（修改：Initialize 末尾增加 player.InitPlayStatePersistence()，Stop 中 player.StopPlayer() 前增加 player.SavePlayState()）
- 影响范围：播放器重启后自动恢复播放进度；不影响原有播放、切歌、列表管理逻辑
- 是否需要重放补丁：是。原作者更新后需重新应用以下入口改动：
  - internal.go 中 Initialize() 末尾增加 player.InitPlayStatePersistence() 调用
  - internal.go 中 Stop() 内增加 player.SavePlayState() 调用
  - state_persist.go 完整放回 internal/player/ 目录
- 回滚方式：删除 state_persist.go，还原 internal.go 中被修改的两行

## 2026-06-27：连接前预断开

- 日期：2026-06-27
- 目标：点击连接按钮时先执行断开操作再连接，避免 SDK 状态残留导致连接异常
- 变更内容：
  - registerHandlers() 中 CmdLiveRoomOperation 的连接分支（SetConnect=true）：在 Connect() 之前插入一行 Disconnect() 调用
  - 预断开不调用 scheduler.Stop()，不误杀正在运行的重连调度器
  - 断开分支（SetConnect=false）保持现状，scheduler.Stop() 仍处于注释状态
- 涉及文件：
  - internal/liveroom/liveroom.go（修改：连接分支增加预断开调用）
- 影响范围：用户手动点击连接时的行为；不影响断线重连调度器
- 是否需要重放补丁：是。原作者更新后需在连接分支 Connect() 前重新插入 Disconnect() 调用
- 回滚方式：删除连接分支中的 `room.room.Disconnect()` 行

## 2026-06-28：播放状态持久化诊断日志

- 日期：2026-06-28
- 目标：为播放状态持久化功能增加详细诊断日志，用于排查"启动恢复时好时坏"的初始化时序问题
- 变更内容：
  - 所有日志统一使用 player logger 前缀 `PlayState`，保存/恢复阶段分别加 `[Save]` / `[Restore]` 标签
  - **保存时**：记录触发来源（切歌/列表更新/停止/关闭）、当前歌曲标题、待播列表歌曲数、JSON 序列化耗时、写入文件路径与大小
  - **恢复时**：记录状态文件路径与大小、待播列表每首歌标题、恢复耗时、sync.Once 触发时间点与 EventBus 运行状态
  - **初始化时序**（internal.go + app/main.go）：Initialize() 每一步加上 `[InitTimeline]` 标签，记录启动时间与每步累计耗时；EventBus.Start() 前后也加时序日志
  - eventbus.Controller 接口新增 `IsRunning()` 方法，bus 实现返回 `started && !stopping`
  - 新增 `fmtPathShort()` 辅助函数，截断过长路径用于日志展示
- 涉及文件：
  - internal/player/state_persist.go（修改：InitPlayStatePersistence / SavePlayState / savePlayStateInternal / readPlayState 全部增加诊断日志）
  - internal/internal.go（修改：Initialize() 增加每步时序日志）
  - app/main.go（修改：EventBus.Start() 前后增加时序日志）
  - pkg/eventbus/bus.go（修改：Controller 接口增加 IsRunning()）
  - pkg/eventbus/bus_impl.go（修改：bus 实现 IsRunning()）
- 影响范围：仅增加日志输出和接口方法，不改变任何业务行为
- 是否需要重放补丁：是。原作者更新后需重新应用所有日志增加的代码行及 IsRunning() 接口方法
- 回滚方式：还原各文件中被修改的段落，删除 IsRunning() 方法定义与实现

## 2026-06-29：修复播放状态恢复的文件锁冲突和时序混乱

- 日期：2026-06-29
- 目标：修复重启后播放状态恢复的两个 Bug：文件锁冲突（Windows 下 `os.Remove` 与 `os.WriteFile` 并发访问同一文件报错）和恢复时序混乱（待播列表先恢复被 Controller 逐步消费殆尽，导致保存 0 首歌的空状态）
- 变更内容：
  - **修复1**：`removePlayStateFile()` 加 `stateFileMutex.Lock()/Unlock()`，与 `savePlayStateInternal()` 互斥
  - **修复2**：状态文件延迟删除。恢复阶段不再调用 `removePlayStateFile()`，仅在 `savePlayStateInternal()` 写文件前删除旧文件（已有 mutex 保护，安全）。恢复完成仅清空 `pendingRestore = nil`
  - **修复3**：调整恢复顺序为先恢复当前歌曲（`PlayerPlayCmd` 发布），再恢复待播列表。移除 `sync.Once` + `PlayerPropertyStateUpdate` 订阅方式，改为直接在 `InitPlayStatePersistence()` 中同步发布。这样当前歌曲不进入待播列表，Controller 消费时不会吃掉正在恢复的歌曲
  - 移除 `sync.Once` 变量 `restoreOnce` 和 `runtime` import
- 涉及文件：
  - internal/player/state_persist.go（修改：InitPlayStatePersistence 重构恢复流程；savePlayStateInternal 增加延迟删除；removePlayStateFile 加锁）
- 影响范围：重启后播放状态恢复行为更可靠；保存与删除操作不再出现文件锁冲突
- 是否需要重放补丁：是。原作者更新后需完整替换 state_persist.go
- 回滚方式：还原 state_persist.go 到 2026-06-28 版本

## 2026-06-29：修复初始化阶段播放状态保存泛滥

- 日期：2026-06-29
- 目标：修复初始化完成前所有中间播放状态被保存的问题——恢复的播放列表被 Controller 消费、系统播放列表加载触发的大量 PlaylistDetailUpdate 事件，均导致 play_state.json 被反复写入，最终可能保存空状态（pending_songs=0）
- 变更内容：
  - 新增 `saveEnabled atomic.Bool` 包级变量（默认 false），作为保存操作的全局开关
  - `savePlayStateInternal()` 开头增加 `if !saveEnabled.Load() { return }` 守卫，初始化期间直接跳过所有保存
  - `InitPlayStatePersistence()` 末尾（所有恢复操作 + 事件订阅完成后）调用 `saveEnabled.Store(true)` 启用保存
  - 新增 `sync/atomic` import
- 涉及文件：
  - internal/player/state_persist.go（修改：新增 saveEnabled 变量、savePlayStateInternal 守卫、InitPlayStatePersistence 末尾启用）
- 影响范围：重启后播放状态恢复期间不再写入中间状态文件；用户正常使用后的切歌、列表增删保存行为不受影响；关闭时的 SavePlayState() 不受影响（saveEnabled 已为 true）
- 是否需要重放补丁：是。原作者更新后需完整替换 state_persist.go
- 回滚方式：还原 state_persist.go 到上一版本

## 2026-06-29：修复播放器就绪等待与当前歌曲保存恢复

- 日期：2026-06-29
- 目标：修复播放状态持久化的两个问题：1) 等待播放器就绪后再恢复，避免恢复过早导致播放失败跳过歌曲；2) 保存时包含当前正在播放的歌曲，恢复后通过执行"下一首"来续播
- 变更内容：
  - **问题1**：恢复流程从固定 5 秒延迟改为"5 秒初始延迟 + 等待 PlayerPlayingUpdate.Removed=false 信号"。5 秒后订阅 PlayerPlayingUpdate，用 channel 等待播放器开始播放的信号，30 秒超时后放弃等待。收到信号或超时后取消订阅再执行恢复
  - **问题2**：新增 `currentSong` + `currentSongLock` 变量，通过订阅 PlayerPlayCmd 事件追踪当前播放歌曲。SavePlayState 保存 currentSong 而非空的 model.Media{}。恢复时合并 currentSong + 待播列表，一次性全部插入 PlayerPlaylist，设 Index=0，发布 PlayerPlayNextCmd 从当前歌曲开始续播
  - 移除旧的 PlayerPlayCmd 直接发布方式，恢复统一走 PlayerPlaylist → PlayerPlayNextCmd 标准流程
- 涉及文件：
  - internal/player/state_persist.go（修改：InitPlayStatePersistence 增加播放器就绪等待；SavePlayState 增加当前歌曲追踪与保存；新增 currentSong/currentSongLock 变量和 PlayerPlayCmd 订阅）
- 影响范围：重启后播放状态恢复更可靠，不会再出现恢复过早导致播放失败跳过歌曲的问题；关闭时保存的当前歌曲可被恢复
- 是否需要重放补丁：是。原作者更新后需完整替换 state_persist.go
- 回滚方式：还原 state_persist.go 到 "重构播放状态持久化 — 极简化" 版本

## 2026-06-29：重构播放状态持久化 — 极简化

- 日期：2026-06-29
- 目标：大幅简化播放状态持久化逻辑。启动时延迟 5 秒恢复，运行时不做任何保存，仅在关闭时保存。删除 saveEnabled 标志和所有事件驱动的保存逻辑。
- 变更内容：
  - 删除 `saveEnabled atomic.Bool` 变量及 `sync/atomic` import
  - 删除所有事件驱动的保存订阅（PlayerPlayCmd、PlaylistDetailUpdate、PlayerPlayingUpdate 三个订阅及对应的 savePlayStateInternal 调用）
  - 删除 `currentSong` / `currentSongLock`（不再运行时追踪当前歌曲）
  - 删除 `savePlayStateInternal`、`fmtPathShort`、`removePlayStateFile` 三个辅助函数
  - `InitPlayStatePersistence` 简化为：读取状态文件 → 有则延迟 5 秒恢复（先发布 PlayerPlayCmd 恢复当前歌曲，再填充 PlayerPlaylist，最后删除状态文件）→ 无则不做事
  - `SavePlayState` 简化为：从 PlayerPlaylist 获取待播列表 → 序列化写入 play_state.json → 打印详情日志。shutdown 时当前歌曲已停止，仅保存待播列表
  - 文件最终仅保留三个函数：`InitPlayStatePersistence`、`SavePlayState`、`readPlayState`，以及 `stateFileMutex`
- 涉及文件：
  - internal/player/state_persist.go（完整重写，从 215 行缩减到 ~130 行）
- 影响范围：播放状态仅在两处触发持久化——启动时延迟恢复、关闭时保存。运行时不再有任何磁盘 I/O
- 是否需要重放补丁：是。原作者更新后需完整替换 state_persist.go
- 回滚方式：还原 state_persist.go 到上一版本（即 "修复初始化阶段播放状态保存泛滥" 版本）

## 2026-06-29：修复播放器就绪信号丢失导致恢复超时

- 日期：2026-06-29
- 目标：修复重启恢复时永远等待 30 秒超时的问题——歌曲已在 10 秒内开始播放，但 PlayerPlayingUpdate.Removed=false 信号在订阅建立前已发出，订阅建立后永远收不到
- 变更内容：
  - 将 readyCh 和 PlayerPlayingUpdate 订阅从 time.AfterFunc(10s) 内部提前到外部，在延迟期间即可捕获信号
  - 延迟结束后先非阻塞 select 检查 channel 是否已有信号，有则直接恢复，无则进入 30 秒等待
  - 初始延迟从 5 秒改为 10 秒，给播放器更充裕的初始化时间
  - 更新 customization-plan.md 第 12 节规则描述
- 涉及文件：
  - internal/player/state_persist.go（修改：InitPlayStatePersistence 中订阅/等待逻辑重构）
  - docs/customization-plan.md（修改：12.2 规则描述更新）
- 影响范围：重启后恢复响应更快，不再因错过信号而空等 30 秒
- 是否需要重放补丁：是。原作者更新后需完整替换 state_persist.go
- 回滚方式：还原 state_persist.go 到前两版本

## 2026-06-29：持久化函数收敛到 auto_actions.go

- 日期：2026-06-29
- 目标：将 player.InitPlayStatePersistence 和 player.SavePlayState 调用收敛到 auto_actions.go，internal.go 不再直接引用 player 包的持久化函数
- 变更内容：
  - auto_actions.go：InitAutoActions() 内部新增 player.InitPlayStatePersistence() 调用；新增 SaveAutoActions() 函数包装 player.SavePlayState()
  - internal.go：Initialize() 中移除 player.InitPlayStatePersistence() 独立调用，仅保留 InitAutoActions()；Stop() 中 player.SavePlayState() 替换为 SaveAutoActions()
  - internal.go 步骤编号重新排序：Step 5 统一为 InitAutoActions，移除 5b
- 涉及文件：
  - internal/auto_actions.go（修改：InitAutoActions 增加持久化调用，新增 SaveAutoActions）
  - internal/internal.go（修改：移除 player.InitPlayStatePersistence 和 player.SavePlayState 独立调用）
- 影响范围：播放状态持久化功能入口收敛，internal.go 不再直接依赖 player 持久化 API
- 是否需要重放补丁：是。原作者更新后需在 internal.go 的 Initialize 和 Stop 中分别调用 InitAutoActions() 和 SaveAutoActions()
- 回滚方式：还原 internal.go 和 auto_actions.go 到本次修改前版本
