package wsTaskInfo

import (
	"github.com/kubackup/kubackup/internal/consts"
	"io"
	"sync"
	"time"
)

// MaxErrorNum 最大存储数据数量，配合limitNum使用
const MaxErrorNum = 10

type Sprintf struct {
	Sprints        []*Sprint
	taskInfo       WsTaskInfo
	MinUpdatePause time.Duration
	lastUpdate     time.Time
	limitNum       int
	limitNumLock   sync.Mutex
	io.Writer
}

type Sprint struct {
	Clear bool   `json:"clear"`
	Text  string `json:"text"`
	Time  string `json:"time"`
	Level int    `json:"level"`
}

func NewSprintf(task WsTaskInfo) *Sprintf {
	return &Sprintf{
		taskInfo:       task,
		Sprints:        make([]*Sprint, 0),
		MinUpdatePause: time.Second,
		limitNum:       0,
	}
}
func (sf *Sprintf) UpdateTaskInfo(task WsTaskInfo) {
	sf.taskInfo = task
}
func (sf *Sprintf) SetMinUpdatePause(d time.Duration) {
	sf.MinUpdatePause = d
}

// ResetLimitNum 重置日志最大条数限制
func (sf *Sprintf) ResetLimitNum() {
	sf.limitNumLock.Lock()
	defer sf.limitNumLock.Unlock()
	sf.limitNum = 0
}

func (sf *Sprintf) Append(level int, str string) {
	sf.AppendByForce(level, str, true)
}

func (sf *Sprintf) AppendByForce(level int, str string, force bool) {
	s := newSprint(level, str, false)
	sf.send(s, force)
	sf.Sprints = append(sf.Sprints, s)
}

func (sf *Sprintf) AppendLimit(level int, str string) {
	sf.limitNumLock.Lock()
	defer sf.limitNumLock.Unlock()
	if sf.limitNum < MaxErrorNum {
		sf.AppendByForce(level, str, false)
	}
	sf.limitNum++
}

func (sf *Sprintf) AppendForClear(level int, str string, save bool) {
	s := newSprint(level, str, true)
	sf.send(s, false)
	if save {
		sf.Sprints = append(sf.Sprints, s)
	}
}

func (sf *Sprintf) SendAllLog() {
	for i, s := range sf.Sprints {
		if i == len(sf.Sprints)-1 {
			sf.send(s, true)
			continue
		}
		if s.Clear && sf.Sprints[i+1].Clear {
			continue
		}
		sf.send(s, true)
	}
}

func newSprint(level int, str string, clear bool) *Sprint {
	s := &Sprint{}
	s.Text = str
	s.Time = time.Now().Format(consts.Custom)
	s.Level = level
	s.Clear = clear
	return s
}

func (sf *Sprintf) send(s interface{}, force bool) {
	if !force && (time.Since(sf.lastUpdate) < sf.MinUpdatePause || sf.MinUpdatePause == 0) {
		return
	}
	sf.lastUpdate = time.Now()
	sf.taskInfo.SendMsg(s)
}

func (sf *Sprintf) Write(p []byte) (n int, err error) {
	text := string(p)
	s := &Sprint{}
	s.Text = text
	sf.send(s, false)
	sf.Sprints = append(sf.Sprints, s)
	return len(text), nil
}

const (
	Info    = 1
	Warning = 2
	Success = 3
	Error   = 4
)
