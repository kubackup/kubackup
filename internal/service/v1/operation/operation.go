package operation

import (
	"github.com/asdine/storm/v3/q"
	"github.com/kubackup/kubackup/internal/entity/v1/operation"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"time"
)

type Service interface {
	common.DBService
	Create(operation *operation.Operation, options common.DBOptions) error
	List(repoid, optype int, options common.DBOptions) ([]operation.Operation, error)
	ListLast(repoid, optype int, options common.DBOptions) (operation.Operation, error)
	Update(operation *operation.Operation, options common.DBOptions) error
	UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error
}

func GetService() Service {
	return &Operation{
		DefaultDBService: common.DefaultDBService{},
	}
}

type Operation struct {
	common.DefaultDBService
}

func (o Operation) Update(operation *operation.Operation, options common.DBOptions) error {
	db := o.GetDB(options)
	operation.UpdatedAt = time.Now()
	return db.Update(operation)
}

func (o Operation) UpdateField(id int, fieldName string, value interface{}, options common.DBOptions) error {
	db := o.GetDB(options)
	th := &operation.Operation{}
	th.Id = id
	th.UpdatedAt = time.Now()
	return db.UpdateField(th, fieldName, value)
}

func (o Operation) Create(operation *operation.Operation, options common.DBOptions) error {
	db := o.GetDB(options)
	operation.CreatedAt = time.Now()
	return db.Save(operation)
}

func (o Operation) List(repoid, optype int, options common.DBOptions) (operations []operation.Operation, err error) {
	db := o.GetDB(options)
	operations = make([]operation.Operation, 0)
	var ms []q.Matcher
	if repoid > 0 {
		ms = append(ms, q.Eq("RepositoryId", repoid))
	}
	if optype > 0 {
		ms = append(ms, q.Eq("Type", optype))
	}
	query := db.Select(q.And(ms...)).OrderBy("CreatedAt").Reverse()
	if err = query.Find(&operations); err != nil {
		return
	}
	return
}

func (o Operation) ListLast(repoid, optype int, options common.DBOptions) (operations operation.Operation, err error) {
	db := o.GetDB(options)
	operations = operation.Operation{}
	var ms []q.Matcher
	ms = append(ms, q.Eq("RepositoryId", repoid))
	ms = append(ms, q.Eq("Type", optype))
	query := db.Select(q.And(ms...)).OrderBy("CreatedAt").Reverse().Limit(1)
	if err = query.First(&operations); err != nil {
		return
	}
	return
}
