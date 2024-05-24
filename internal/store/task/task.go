package task

import (
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/utils"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"sync"
)

// 任务状态
const (
	StatusNew     = 0 //新建
	StatusRunning = 1 //运行中
	StatusEnd     = 2 //已完成
	StatusError   = 3 //错误
)

var TaskInfos = &TaskMap{TaskInfos: make(map[int]wsTaskInfo.WsTaskInfo)}

type TaskInfo struct {
	id            int
	bound         chan error
	sockJSSession sockjs.Session
	Name          string
	Path          string
	Progress      *model.StatusUpdate
	wsTaskInfo.WsTaskInfo
}

func (ti *TaskInfo) GetId() int {
	return ti.id
}
func (ti *TaskInfo) SetId(id int) {
	ti.id = id
}
func (ti *TaskInfo) SetBound(c chan error) {
	ti.bound = c
}
func (ti *TaskInfo) IntoBound(msg error) {
	if ti.bound != nil {
		ti.bound <- msg
	}
}
func (ti *TaskInfo) GetBound() chan error {
	return ti.bound
}
func (ti *TaskInfo) SetSockJSSession(s sockjs.Session) {
	ti.sockJSSession = s
}
func (ti *TaskInfo) SendMsg(msg interface{}) {
	if ti.sockJSSession != nil && ti.sockJSSession.ID() != "" {
		go func() {
			err := ti.sockJSSession.Send(utils.ToJSONString(msg))
			if err != nil {
				ti.SetSockJSSession(nil)
			}
		}()
	}
}

func (ti *TaskInfo) CloseSockJSSession(reason string, status uint32) {
	if ti.sockJSSession != nil {
		err := ti.sockJSSession.Close(status, reason)
		if err != nil {
			server.Logger().Error(err)
		}
	}
}

func (ti *TaskInfo) CloseBound() {
	if ti.bound != nil {
		close(ti.bound)
	}
}

type TaskMap struct {
	TaskInfos map[int]wsTaskInfo.WsTaskInfo
	Lock      sync.Mutex
	wsTaskInfo.WsTask
}

func (ti *TaskMap) Get(id int) wsTaskInfo.WsTaskInfo {
	ti.Lock.Lock()
	defer ti.Lock.Unlock()
	return ti.TaskInfos[id]
}
func (ti *TaskMap) Set(id int, task wsTaskInfo.WsTaskInfo) {
	ti.Lock.Lock()
	defer ti.Lock.Unlock()
	ti.TaskInfos[id] = task
}

func (ti *TaskMap) Close(id int, reason string, status uint32) {
	ti.Lock.Lock()
	defer ti.Lock.Unlock()
	ti.TaskInfos[id].CloseSockJSSession(reason, status)
	ti.TaskInfos[id].CloseBound()
	delete(ti.TaskInfos, id)
}

// GetCount 获取进行中任务数量
func (ti *TaskMap) GetCount() int {
	return len(ti.TaskInfos)
}
