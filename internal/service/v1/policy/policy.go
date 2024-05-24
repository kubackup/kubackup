package policy

import (
	"fmt"
	"github.com/asdine/storm/v3/q"
	"github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"time"
)

type Service interface {
	common.DBService
	Create(policy *repository.ForgetPolicy, options common.DBOptions) error
	List(options common.DBOptions) ([]repository.ForgetPolicy, error)
	Search(repoId int, path string, options common.DBOptions) ([]repository.ForgetPolicy, error)
	Get(id int, options common.DBOptions) (*repository.ForgetPolicy, error)
	Delete(id int, options common.DBOptions) error
	Update(policy *repository.ForgetPolicy, options common.DBOptions) error
	UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error
}

func GetService() Service {
	return &Policy{
		DefaultDBService: common.DefaultDBService{},
	}
}

type Policy struct {
	common.DefaultDBService
}

func (p Policy) List(options common.DBOptions) (policies []repository.ForgetPolicy, err error) {
	db := p.GetDB(options)
	policies = make([]repository.ForgetPolicy, 0)
	var ms []q.Matcher
	query := db.Select(ms...).OrderBy("CreatedAt").Reverse()
	if err = query.Find(&policies); err != nil {
		return
	}
	return
}

func (p Policy) Create(policy *repository.ForgetPolicy, options common.DBOptions) error {
	db := p.GetDB(options)
	policies, err := p.Search(policy.RepositoryId, policy.Path, options)
	if err != nil && err.Error() != "not found" {
		return err
	}
	if len(policies) > 0 {
		return fmt.Errorf("数据 %d,%s 已存在", policy.RepositoryId, policy.Path)
	}
	policy.CreatedAt = time.Now()
	return db.Save(policy)
}

func (p Policy) Search(repoId int, path string, options common.DBOptions) (policies []repository.ForgetPolicy, err error) {
	db := p.GetDB(options)
	policies = make([]repository.ForgetPolicy, 0)
	var ms []q.Matcher
	if repoId > 0 {
		ms = append(ms, q.Eq("RepositoryId", repoId))
	}
	if path != "" {
		ms = append(ms, q.Eq("Path", path))
	}
	query := db.Select(ms...).OrderBy("CreatedAt").Reverse()
	if err = query.Find(&policies); err != nil {
		return
	}
	return
}

func (p Policy) Get(id int, options common.DBOptions) (*repository.ForgetPolicy, error) {
	db := p.GetDB(options)
	var rep repository.ForgetPolicy
	err := db.One("Id", id, &rep)
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (p Policy) Delete(id int, options common.DBOptions) error {
	db := p.GetDB(options)
	rep, err := p.Get(id, options)
	if err != nil {
		return err
	}
	return db.DeleteStruct(rep)
}

func (p Policy) Update(policy *repository.ForgetPolicy, options common.DBOptions) error {
	db := p.GetDB(options)
	policy.UpdatedAt = time.Now()
	return db.Update(policy)
}

func (p Policy) UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error {
	db := p.GetDB(options)
	th := &repository.ForgetPolicy{}
	th.Id = id
	th.UpdatedAt = time.Now()
	return db.UpdateField(th, fieldName, value)
}
