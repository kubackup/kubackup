package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/debug"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/filter"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/walker"
	"gopkg.in/tomb.v2"
	"strings"
	"time"
)

func RunFind(targetP string, repoid int, snapshotid string) (*LsRes, error) {
	if snapshotid == "" {
		return nil, errors.Errorf("no snapshot ID specified")
	}
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(repoHandler.gopts.ctx)
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	defer clean.Cleanup()

	var id restic.ID

	if snapshotid == "latest" {
		id, err = restic.FindLatestSnapshot(ctx, repoHandler.repo, []string{}, []restic.TagList{}, []string{}, nil)
		if err != nil {
			return nil, fmt.Errorf("Ignoring %q, no snapshot matched given filter (Paths:%v Tags:%v Hosts:%v)\n", snapshotid, []string{}, []restic.TagList{}, []string{})
		}
	} else {
		id, err = restic.FindSnapshot(ctx, repoHandler.repo, snapshotid)
		if err != nil {
			return nil, errors.Errorf("Ignoring %q: %v\n", snapshotid, err)
		}
	}
	sn, err := restic.LoadSnapshot(ctx, repoHandler.repo, id)
	if err != nil {
		return nil, errors.Errorf("Ignoring %q, could not load snapshot: %v\n", id, err)
	}
	snapshot := lsSnapshot{
		Snapshot:   sn,
		ID:         sn.ID(),
		ShortID:    sn.ID().Str(),
		StructType: "snapshot",
	}
	if sn.Tree == nil {
		return nil, fmt.Errorf("snapshot 404")
	}
	lsres := LsRes{}
	lsres.Snapshot = snapshot
	tree, err := repoHandler.repo.LoadTree(ctx, *sn.Tree)
	if err != nil {
		server.Logger().Error(err)
		return nil, fmt.Errorf("loadIndexing")
	}
	res, err := walk(ctx, repoHandler.repo, "/", tree, snapshot.Paths[0]+"/"+targetP)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		start, _ := parseTime("")
		end, _ := parseTime("")
		res, err = findInSnapshot(ctx, repoHandler.repo, *sn.Tree, targetP, start, end)
		if err != nil {
			return nil, err
		}
	}
	lsres.Nodes = res
	return &lsres, nil
}

var timeFormats = []string{
	"2006-01-02",
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05 MST",
	"02.01.2006",
	"02.01.2006 15:04",
	"02.01.2006 15:04:05",
	"02.01.2006 15:04:05 -0700",
	"02.01.2006 15:04:05 MST",
	"Mon Jan 2 15:04:05 -0700 MST 2006",
}

func parseTime(str string) (time.Time, error) {
	for _, fmts := range timeFormats {
		if t, err := time.ParseInLocation(fmts, str, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.Fatalf("unable to parse time: %q", str)
}

func findInSnapshot(ctx context.Context, repo restic.TreeLoader, tree restic.ID, targetF string, start time.Time, end time.Time) (res []interface{}, err error) {
	res = make([]interface{}, 0)
	targetF = strings.ToLower(targetF)
	parttern := []string{targetF}
	ch := make(chan interface{})
	defer close(ch)
	var t tomb.Tomb
	t.Go(func() error {
		_ = walker.Walk(ctx, repo, tree, restic.NewIDSet(), findres(t.Context(ctx), ch, parttern, start, end))
		t.Kill(nil)
		return nil
	})
	t.Go(func() error {
		for node := range ch {
			res = append(res, node)
			select {
			case <-t.Context(ctx).Done():
				return nil
			}
		}
		return nil
	})
	err = t.Wait()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func findres(ctx context.Context, ch chan interface{}, parttern []string, start time.Time, end time.Time) walker.WalkFunc {
	return func(parentTreeID restic.ID, nodepath string, node *restic.Node, err error) (bool, error) {
		if err != nil {
			debug.Log("Error loading tree %v: %v", parentTreeID, err)
			return false, walker.ErrSkipNode
		}

		if node == nil {
			return false, nil
		}

		normalizedNodepath := nodepath
		normalizedNodepath = strings.ToLower(nodepath)

		var foundMatch bool

		for _, pat := range parttern {
			found, err := filter.Match(pat, normalizedNodepath)
			if err != nil {
				return false, err
			}
			if found {
				foundMatch = true
				break
			}
		}

		var (
			ignoreIfNoMatch = true
			errIfNoMatch    error
		)
		if node.Type == "dir" {
			var childMayMatch bool
			for _, pat := range parttern {
				mayMatch, err := filter.ChildMatch(pat, normalizedNodepath)
				if err != nil {
					return false, err
				}
				if mayMatch {
					childMayMatch = true
					break
				}
			}

			if !childMayMatch {
				ignoreIfNoMatch = true
				errIfNoMatch = walker.ErrSkipNode
			} else {
				ignoreIfNoMatch = false
			}
		}

		if !foundMatch {
			return ignoreIfNoMatch, errIfNoMatch
		}

		if !start.IsZero() && node.ModTime.Before(start) {
			return ignoreIfNoMatch, errIfNoMatch
		}

		if !end.IsZero() && node.ModTime.After(end) {
			return ignoreIfNoMatch, errIfNoMatch
		}
		select {
		case ch <- getNode(nodepath, node):
		case <-ctx.Done():
			return false, ctx.Err()
		}
		return false, nil
	}
}
