package task

import (
	"github.com/asdine/storm/v3/q"
	"github.com/kubackup/kubackup/internal/entity/v1/task"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/pkg/storm"
	"time"
)

type Service interface {
	common.DBService
	Create(task *task.Task, options common.DBOptions) error
	Search(num, size, status, RepositoryId, planId int, path, name string, options common.DBOptions) (int, []task.Task, error)
	List(num, size int, options common.DBOptions) (int, []task.Task, error)
	Get(id int, options common.DBOptions) (*task.Task, error)
	Update(task *task.Task, options common.DBOptions) error
	UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error
}

func GetService() Service {
	return &Task{
		DefaultDBService: common.DefaultDBService{},
	}
}

type Task struct {
	common.DefaultDBService
}

func (t Task) Search(num, size, status, RepositoryId, planId int, path, name string, options common.DBOptions) (total int, res []task.Task, err error) {
	res = []task.Task{}
	db := t.GetDB(options)
	var ms []q.Matcher
	if status >= 0 {
		ms = append(ms, q.Eq("Status", status))
	}
	if RepositoryId > 0 {
		ms = append(ms, q.Eq("RepositoryId", RepositoryId))
	}
	if planId > 0 {
		ms = append(ms, q.Eq("PlanId", planId))
	}
	if path != "" {
		ms = append(ms, storm.Like("Path", path))
	}
	if name != "" {
		ms = append(ms, storm.Like("Name", name))
	}
	query := db.Select(ms...).OrderBy("CreatedAt").Reverse()
	total, err = query.Count(&task.Task{})
	if err != nil {
		_ = db.Drop(&task.Task{})
		return
	}
	if size != 0 {
		query.Limit(size).Skip((num - 1) * size)
	}
	if err = query.Find(&res); err != nil {
		return
	}
	return
}

func (t Task) UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error {
	db := t.GetDB(options)
	th := &task.Task{}
	th.Id = id
	th.UpdatedAt = time.Now()
	return db.UpdateField(th, fieldName, value)
}

func (t Task) Create(task *task.Task, options common.DBOptions) error {
	db := t.GetDB(options)
	task.CreatedAt = time.Now()
	return db.Save(task)
}

func (t Task) List(num, size int, options common.DBOptions) (total int, res []task.Task, err error) {
	res = make([]task.Task, 0)
	db := t.GetDB(options)
	query := db.Select().OrderBy("CreatedAt").Reverse()
	total, err = query.Count(&task.Task{})
	if err != nil {
		return
	}
	if size != 0 {
		query.Limit(size).Skip((num - 1) * size)
	}
	if err = query.Find(&res); err != nil {
		return
	}
	return
}

func (t Task) Get(id int, options common.DBOptions) (*task.Task, error) {
	db := t.GetDB(options)
	var rep task.Task
	err := db.One("Id", id, &rep)
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (t Task) Update(task *task.Task, options common.DBOptions) error {
	db := t.GetDB(options)
	task.UpdatedAt = time.Now()
	return db.Update(task)
}
