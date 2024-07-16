package log

import (
	"github.com/kubackup/kubackup/internal/server"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/utils"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"sync"
)

var LogInfos = &LogMap{TaskInfos: make(map[int]wsTaskInfo.WsTaskInfo)}

type LogInfo struct {
	id            int
	bound         chan string
	sockJSSession sockjs.Session
	wsTaskInfo.WsTaskInfo
}

func (ti *LogInfo) GetId() int {
	return ti.id
}
func (ti *LogInfo) SetId(id int) {
	ti.id = id
}
func (ti *LogInfo) SetBound(c chan string) {
	ti.bound = c
}
func (ti *LogInfo) IntoBound(msg string) {
	if ti.bound != nil {
		ti.bound <- msg
	}
}
func (ti *LogInfo) GetBound() chan string {
	return ti.bound
}
func (ti *LogInfo) CloseBound() {
	if ti.bound != nil {
		close(ti.bound)
	}
}
func (ti *LogInfo) SetSockJSSession(s sockjs.Session) {
	ti.sockJSSession = s
}
func (ti *LogInfo) SendMsg(msg interface{}) {
	if ti.sockJSSession != nil && ti.sockJSSession.ID() != "" && msg != "" {
		go func() {
			err := ti.sockJSSession.Send(utils.ToJSONString(msg))
			if err != nil {
				ti.SetSockJSSession(nil)
			}
		}()
	}
}

func (ti *LogInfo) CloseSockJSSession(reason string, status uint32) {
	if ti.sockJSSession != nil {
		err := ti.sockJSSession.Close(status, reason)
		if err != nil {
			server.Logger().Error(err)
		}
	}
}

type LogMap struct {
	TaskInfos map[int]wsTaskInfo.WsTaskInfo
	Lock      sync.Mutex
	wsTaskInfo.WsTask
}

func (ti *LogMap) Get(id int) wsTaskInfo.WsTaskInfo {
	ti.Lock.Lock()
	defer ti.Lock.Unlock()
	return ti.TaskInfos[id]
}
func (ti *LogMap) Set(id int, task wsTaskInfo.WsTaskInfo) {
	ti.Lock.Lock()
	defer ti.Lock.Unlock()
	ti.TaskInfos[id] = task
}

func (ti *LogMap) Close(id int, reason string, status uint32) {
	ti.Lock.Lock()
	defer ti.Lock.Unlock()
	ti.TaskInfos[id].CloseSockJSSession(reason, status)
	ti.TaskInfos[id].CloseBound()
	delete(ti.TaskInfos, id)
}

// GetCount 获取进行中任务数量
func (ti *LogMap) GetCount() int {
	return len(ti.TaskInfos)
}
