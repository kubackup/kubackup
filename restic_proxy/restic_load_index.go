package resticProxy

import (
	"context"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
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
	if err = LoadIndex(ctx, repo); err != nil {
		return err
	}
	return nil
}

func LoadIndex(ctx context.Context, r *repository.Repository) error {
	midx := repository.NewMasterIndex()
	validIndex := restic.NewIDSet()
	err := repository.ForAllIndexes(ctx, r, func(id restic.ID, idx *repository.Index, oldFormat bool, err error) error {
		if err != nil {
			return err
		}

		ids, err := idx.IDs()
		if err != nil {
			return err
		}

		for _, id := range ids {
			validIndex.Insert(id)
		}
		midx.Insert(idx)
		return nil
	})

	if err != nil {
		return errors.Fatal(err.Error())
	}
	err = midx.MergeFinalIndexes()
	if err != nil {
		return err
	}
	err = r.SetIndex(midx)
	if err != nil {
		return err
	}
	// remove index files from the cache which have been removed in the repo
	return r.PrepareCache(validIndex)
}
