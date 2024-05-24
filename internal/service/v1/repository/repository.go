package repository

import (
	"github.com/asdine/storm/v3/q"
	"github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/pkg/storm"
	"time"
)

type Service interface {
	common.DBService
	Create(repository *repository.Repository, options common.DBOptions) error
	List(repotype int, name string, options common.DBOptions) ([]repository.Repository, error)
	Get(id int, options common.DBOptions) (*repository.Repository, error)
	Delete(id int, options common.DBOptions) error
	Update(repository *repository.Repository, options common.DBOptions) error
	UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error
}

func GetService() Service {
	return &Repository{
		DefaultDBService: common.DefaultDBService{},
	}
}

type Repository struct {
	common.DefaultDBService
}

func (c *Repository) UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error {
	db := c.GetDB(options)
	th := &repository.Repository{}
	th.Id = id
	th.UpdatedAt = time.Now()
	return db.UpdateField(th, fieldName, value)
}

func (c *Repository) Get(id int, options common.DBOptions) (*repository.Repository, error) {
	db := c.GetDB(options)
	var rep repository.Repository
	err := db.One("Id", id, &rep)
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (c *Repository) Update(repository *repository.Repository, options common.DBOptions) error {
	db := c.GetDB(options)
	repository.UpdatedAt = time.Now()
	return db.Update(repository)
}

func (c *Repository) Delete(id int, options common.DBOptions) error {
	db := c.GetDB(options)
	rep, err := c.Get(id, options)
	if err != nil {
		return err
	}
	return db.DeleteStruct(rep)
}

func (c *Repository) Create(repository *repository.Repository, options common.DBOptions) error {
	db := c.GetDB(options)
	repository.CreatedAt = time.Now()
	return db.Save(repository)
}

func (c *Repository) List(repotype int, name string, options common.DBOptions) (repositorys []repository.Repository, err error) {
	db := c.GetDB(options)
	repositorys = make([]repository.Repository, 0)
	var ms []q.Matcher
	if repotype > 0 {
		ms = append(ms, q.Eq("Type", repotype))
	}
	if name != "" {
		ms = append(ms, storm.Like("Name", name))
	}
	query := db.Select(ms...).OrderBy("CreatedAt").Reverse()
	if err = query.Find(&repositorys); err != nil {
		return
	}
	return
}
