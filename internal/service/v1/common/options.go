package common

import (
	"github.com/asdine/storm/v3"
	"github.com/kubackup/kubackup/internal/server"
)

type DBService interface {
	GetDB(options DBOptions) storm.Node
}

type DefaultDBService struct {
}

func (d *DefaultDBService) GetDB(options DBOptions) storm.Node {
	if options.DB != nil {
		return options.DB
	}
	return server.DB()
}

type DBOptions struct {
	DB storm.Node
}
