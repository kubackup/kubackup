package resticProxy

import (
	"context"
	"fmt"
	"reflect"
	"unsafe"

	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
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
	workerCount := repo.Connections()

	// 获取底层 backend 用于删除文件
	be := getBackend(repo)

	for i := 0; i < int(workerCount); i++ {
		wg.Go(func() error {
			for id := range fileChan {
				var err error
				if be != nil {
					// 使用 backend 的 Remove 方法删除任意类型的文件
					err = be.Remove(ctx, backend.Handle{Type: fileType, Name: id.String()})
				} else {
					// 回退到使用 RemoveUnpacked（只适用于快照类型）
					err = repo.RemoveUnpacked(ctx, restic.WriteableFileType(fileType), id)
				}
				if err != nil {
					spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("unable to remove %v %v from the repository\n", fileType, id))
					if !ignoreError {
						return err
					}
					spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("remove %v %v error: %v", fileType, id, err))
				}
				pro.Add(1)
			}
			return nil
		})
	}
	err := wg.Wait()
	return err
}

// getBackend 使用反射和 unsafe 获取底层 backend
func getBackend(repo restic.Repository) backend.Backend {
	// 尝试获取底层 *repository.Repository 的 be 字段
	val := reflect.ValueOf(repo)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 查找 be 字段（底层 backend）
	beField := val.FieldByName("be")
	if beField.IsValid() {
		// 使用 unsafe 绕过未导出字段的访问限制
		bePtr := unsafe.Pointer(beField.UnsafeAddr())
		be := (*backend.Backend)(bePtr)
		if *be != nil {
			return *be
		}
	}

	return nil
}
