package plan

import (
	"github.com/asdine/storm/v3/q"
	"github.com/kubackup/kubackup/internal/entity/v1/plan"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/pkg/storm"
	"time"
)

type Service interface {
	common.DBService
	Create(plan *plan.Plan, options common.DBOptions) error
	List(status int, options common.DBOptions) ([]plan.Plan, error)
	Search(num, size, status, RepositoryId int, path, name string, options common.DBOptions) (int, []plan.Plan, error)
	Get(id int, options common.DBOptions) (*plan.Plan, error)
	Delete(id int, options common.DBOptions) error
	Update(plan *plan.Plan, options common.DBOptions) error
	UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error
}

func GetService() Service {
	return &Plan{
		DefaultDBService: common.DefaultDBService{},
	}
}

type Plan struct {
	common.DefaultDBService
}

func (p Plan) Search(num, size, status, RepositoryId int, path, name string, options common.DBOptions) (count int, res []plan.Plan, err error) {
	res = make([]plan.Plan, 0)
	db := p.GetDB(options)
	var ms []q.Matcher
	if status > 0 {
		ms = append(ms, q.Eq("Status", status))
	}
	if RepositoryId > 0 {
		ms = append(ms, q.Eq("RepositoryId", RepositoryId))
	}
	if path != "" {
		ms = append(ms, storm.Like("Path", path))
	}
	if name != "" {
		ms = append(ms, storm.Like("Name", name))
	}
	query := db.Select(ms...).OrderBy("CreatedAt").Reverse()
	count, err = query.Count(&plan.Plan{})
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

func (p Plan) Delete(id int, options common.DBOptions) error {
	db := p.GetDB(options)
	rep, err := p.Get(id, options)
	if err != nil {
		return err
	}
	return db.DeleteStruct(rep)
}

func (p Plan) Create(plan *plan.Plan, options common.DBOptions) error {
	db := p.GetDB(options)
	plan.CreatedAt = time.Now()
	return db.Save(plan)
}

func (p Plan) List(status int, options common.DBOptions) ([]plan.Plan, error) {
	db := p.GetDB(options)
	var ms []q.Matcher
	if status > 0 {
		ms = append(ms, q.Eq("Status", status))
	}
	query := db.Select(ms...).OrderBy("CreatedAt").Reverse()
	repositorys := make([]plan.Plan, 0)
	if err := query.Find(&repositorys); err != nil {
		return nil, err
	}
	return repositorys, nil
}

func (p Plan) Get(id int, options common.DBOptions) (*plan.Plan, error) {
	db := p.GetDB(options)
	var rep plan.Plan
	err := db.One("Id", id, &rep)
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (p Plan) Update(plan *plan.Plan, options common.DBOptions) error {
	db := p.GetDB(options)
	plan.UpdatedAt = time.Now()
	return db.Update(plan)
}

func (p Plan) UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error {
	db := p.GetDB(options)
	th := &plan.Plan{}
	th.Id = id
	th.UpdatedAt = time.Now()
	return db.UpdateField(th, fieldName, value)
}
