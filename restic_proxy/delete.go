package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"golang.org/x/sync/errgroup"
)

// DeleteFiles deletes the given fileList of fileType in parallel
// it will print a warning if there is an error, but continue deleting the remaining files
func DeleteFiles(spr *wsTaskInfo.Sprintf, ctx context.Context, repo restic.Repository, fileList restic.IDSet, fileType restic.FileType) {
	_ = deleteFiles(spr, ctx, true, repo, fileList, fileType)
}

// DeleteFilesChecked deletes the given fileList of fileType in parallel
// if an error occurs, it will cancel and return this error
func DeleteFilesChecked(spr *wsTaskInfo.Sprintf, ctx context.Context, repo restic.Repository, fileList restic.IDSet, fileType restic.FileType) error {
	return deleteFiles(spr, ctx, false, repo, fileList, fileType)
}

const numDeleteWorkers = 8

// deleteFiles deletes the given fileList of fileType in parallel
// if ignoreError=true, it will print a warning if there was an error, else it will abort.
func deleteFiles(spr *wsTaskInfo.Sprintf, ctxx context.Context, ignoreError bool, repo restic.Repository, fileList restic.IDSet, fileType restic.FileType) error {
	totalCount := len(fileList)
	fileChan := make(chan restic.ID)
	wg, ctx := errgroup.WithContext(ctxx)
	wg.Go(func() error {
		defer close(fileChan)
		for id := range fileList {
			select {
			case fileChan <- id:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	pro := newProgressMax(true, uint64(totalCount), "files deleted", spr)
	defer pro.Done()
	spr.ResetLimitNum()
	for i := 0; i < numDeleteWorkers; i++ {
		wg.Go(func() error {
			for id := range fileChan {
				h := restic.Handle{Type: fileType, Name: id.String()}
				err := repo.Backend().Remove(ctx, h)
				if err != nil {
					spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("unable to remove %v from the repository\n", h))
					if !ignoreError {
						return err
					}
				}
				pro.Add(1)
			}
			return nil
		})
	}
	err := wg.Wait()
	return err
}
