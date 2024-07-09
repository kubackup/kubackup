package resticProxy

import (
	"context"
)

func RunLoadIndex(repoid int) error {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return err
	}
	gopts := repoHandler.gopts
	repo := repoHandler.repo
	ctx, cancel := context.WithCancel(gopts.ctx)
	defer cancel()
	err = repo.LoadIndex(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
