package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/task"
	fileutil "github.com/kubackup/kubackup/pkg/file"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/dump"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/utils"
	"os"
	"path"
	"path/filepath"
	"time"
)

// DumpOptions collects all options for the dump command.
type DumpOptions struct {
	Hosts   []string
	Paths   []string
	Tags    restic.TagLists
	Archive string //set archive `format` as "tar" or "zip
}

func splitPath(p string) []string {
	d, f := path.Split(p)
	if d == "" || d == string(filepath.Separator) {
		return []string{f}
	}
	s := splitPath(path.Join(string(filepath.Separator), d))
	return append(s, f)
}

func RunDump(opts DumpOptions, repoid int, snapshotid string, info model.DumpInfo) error {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return err
	}
	repo := repoHandler.repo

	if snapshotid == "" {
		return errors.Fatal("no snapshot ID specified")
	}

	switch opts.Archive {
	case "tar", "zip":
	default:
		return fmt.Errorf("unknown archive format %q", opts.Archive)
	}
	ctx, cancel := context.WithCancel(repoHandler.gopts.ctx)
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})

	snapshotIDString := snapshotid

	splittedPath := splitPath(path.Clean(info.Filename))

	sn, subfolder, err := (&restic.SnapshotFilter{
		Hosts: opts.Hosts,
		Paths: opts.Paths,
		Tags:  opts.Tags,
	}).FindLatest(ctx, repo.Backend(), repo, snapshotid)
	if err != nil {
		return errors.Fatalf("failed to find snapshot: %v", err)
	}
	sn.Tree, err = restic.FindTreeDirectory(ctx, repo, sn.Tree, subfolder)
	if err != nil {
		return err
	}

	tree, err := restic.LoadTree(ctx, repo, *sn.Tree)
	if err != nil {
		clean.Cleanup()
		return fmt.Errorf("loading tree for snapshot %q failed: %v", snapshotIDString, err)
	}
	if splittedPath[0] == "" {
		info.Type = "dir"
	}
	var outfile string
	var fmode os.FileMode
	switch info.Type {
	case "dir":
		outfile = "/tmp/backup/tmp_" + time.Now().String() + "." + opts.Archive
		fmode = 0755
		break
	case "file":
		outfile = info.Filename
		fmode = os.FileMode(info.Mode)
		break
	default:
		clean.Cleanup()
		return fmt.Errorf("unknown type format %q", info.Type)
	}
	outpath := fileutil.GetFilePath(outfile)
	if !fileutil.Exist(outpath) {
		err = os.MkdirAll(outpath, 0777)
		if err != nil {
			clean.Cleanup()
			return err
		}
	}
	w, err := fs.OpenFile(outfile, os.O_CREATE|os.O_RDWR, fmode)
	if err != nil {
		clean.Cleanup()
		return err
	}
	ta, err := createRestoreTask(info.Filename, repoid)
	if err != nil {
		clean.Cleanup()
		return err
	}
	taskinfoid := ta.Id
	go func() {
		defer clean.Cleanup()
		err = taskHistoryService.UpdateField(taskinfoid, "Status", task.StatusRunning, common.DBOptions{})
		if err != nil {
			server.Logger().Error(err)
		}
		start := time.Now()
		d := dump.New(opts.Archive, repo, w)
		err = printFromTree(ctx, tree, repo, "/", info.Type, splittedPath, d)
		if err != nil {
			errorUpdate := model.ErrorUpdate{
				MessageType: "error",
				Error:       err.Error(),
				During:      "restore",
				Item:        info.Filename,
			}
			err1 := taskHistoryService.UpdateField(taskinfoid, "RestoreError", []model.ErrorUpdate{errorUpdate}, common.DBOptions{})
			if err1 != nil {
				server.Logger().Error(err1)
			}
			err = taskHistoryService.UpdateField(taskinfoid, "Status", task.StatusError, common.DBOptions{})
			if err != nil {
				server.Logger().Error(err)
			}
		} else {
			err = taskHistoryService.UpdateField(taskinfoid, "Status", task.StatusEnd, common.DBOptions{})
			if err != nil {
				server.Logger().Error(err)
			}
			summaryOut := &model.SummaryOutput{
				MessageType:         "summary",
				TotalFilesProcessed: 1,
				TotalDuration:       utils.FormatDuration(time.Since(start)),
				SnapshotID:          snapshotid,
			}
			err2 := taskHistoryService.UpdateField(taskinfoid, "Summary", summaryOut, common.DBOptions{})
			if err2 != nil {
				server.Logger().Error(err)
			}
		}
		progress := &model.StatusUpdate{
			MessageType: "status",
			TotalFiles:  1,
			PercentDone: 1,
		}
		err3 := taskHistoryService.UpdateField(taskinfoid, "Progress", progress, common.DBOptions{})
		if err3 != nil {
			server.Logger().Error(err)
		}
	}()
	return nil
}

func printFromTree(ctx context.Context, tree *restic.Tree, repo restic.Repository, prefix, ftype string, pathComponents []string, d *dump.Dumper) error {
	// If we print / we need to assume that there are multiple nodes at that
	// level in the tree.
	if pathComponents[0] == "" {
		if ftype != "dir" {
			return fmt.Errorf("文件类型必须为dir")
		}
		return d.DumpTree(ctx, tree, "/")
	}
	item := filepath.Join(prefix, pathComponents[0])
	l := len(pathComponents)
	for _, node := range tree.Nodes {
		// If dumping something in the highest level it will just take the
		// first item it finds and dump that according to the switch case below.
		if node.Name == pathComponents[0] {
			switch {
			case l == 1 && dump.IsFile(node):
				return d.WriteNode(ctx, node)
			case l > 1 && dump.IsDir(node):
				subtree, err := restic.LoadTree(ctx, repo, *node.Subtree)
				if err != nil {
					return errors.Wrapf(err, "cannot load subtree for %q", item)
				}
				return printFromTree(ctx, subtree, repo, item, ftype, pathComponents[1:], d)
			case dump.IsDir(node):
				if ftype != "dir" {
					return fmt.Errorf("文件类型必须为dir")
				}
				subtree, err := restic.LoadTree(ctx, repo, *node.Subtree)
				if err != nil {
					return err
				}
				return d.DumpTree(ctx, subtree, item)
			case l > 1:
				return fmt.Errorf("%q should be a dir, but is a %q", item, node.Type)
			case !dump.IsFile(node):
				return fmt.Errorf("%q should be a file, but is a %q", item, node.Type)
			}
		}
	}
	return fmt.Errorf("path %q not found in snapshot", item)
}
