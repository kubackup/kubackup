package resticProxy

import (
	"context"
	"encoding/json"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/wxnacy/wgo/arrays"
	"sort"
	"strings"
)

type SnapshotOptions struct {
	restic.SnapshotFilter
	Compact bool
	Last    bool // This option should be removed in favour of Latest.
	Latest  int
	GroupBy restic.SnapshotGroupByOptions
}
type SnapshotRes struct {
	*restic.Snapshot

	ID       *restic.ID              `json:"id"`
	ShortID  string                  `json:"short_id"`
	GroupKey restic.SnapshotGroupKey `json:"group_key"`
}

type SnapshotParm struct {
	Parms []HostParm `json:"parms"`
	Tags  []string   `json:"tags"`
}
type HostParm struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

type filterLastSnapshotsKey struct {
	Hostname    string
	JoinedPaths string
}

func GetParms(repoid int, hosts []string) (*SnapshotParm, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return nil, err
	}
	repo := repoHandler.repo

	ctx, cancel := context.WithCancel(context.Background())
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	defer clean.Cleanup()
	res := &SnapshotParm{}
	tags := make([]string, 0)
	paths := make(map[string][]string, 0)
	for sn := range FindFilteredSnapshots(ctx, repo.Backend(), repo, &restic.SnapshotFilter{Hosts: hosts}, []string{}) {
		ps := paths[sn.Hostname]
		if len(sn.Paths) > 0 && arrays.ContainsString(ps, sn.Paths[0]) < 0 {
			ps = append(ps, sn.Paths...)
		}
		paths[sn.Hostname] = ps
		for _, tag := range sn.Tags {
			if arrays.ContainsString(tags, tag) < 0 {
				tags = append(tags, tag)
			}
		}
	}
	parms := make([]HostParm, 0)
	for k, p := range paths {
		h := HostParm{
			Name:  k,
			Paths: p,
		}
		parms = append(parms, h)
	}
	res.Parms = parms
	res.Tags = tags
	return res, nil
}

func RunSnapshots(opts SnapshotOptions, repoid int, snapshotids []string) ([]interface{}, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return nil, err
	}
	repo := repoHandler.repo

	ctx, cancel := context.WithCancel(context.Background())
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	defer clean.Cleanup()

	var snapshots restic.Snapshots
	for sn := range FindFilteredSnapshots(ctx, repo.Backend(), repo, &opts.SnapshotFilter, snapshotids) {
		snapshots = append(snapshots, sn)
	}
	snapshotGroups, grouped, err := restic.GroupSnapshots(snapshots, opts.GroupBy)
	if err != nil {
		return nil, err
	}

	for k, list := range snapshotGroups {
		if opts.Last {
			// This branch should be removed in the same time
			// that --last.
			list = FilterLastestSnapshots(list, 1)
		} else if opts.Latest > 0 {
			list = FilterLastestSnapshots(list, opts.Latest)
		}
		sort.Sort(list)
		snapshotGroups[k] = list
	}
	var snapshotres []interface{}
	for k, list := range snapshotGroups {
		var key restic.SnapshotGroupKey
		if grouped {
			err = json.Unmarshal([]byte(k), &key)
			if err != nil {
				return nil, err
			}
		}
		for _, sn := range list {
			k := SnapshotRes{
				Snapshot: sn,
				ID:       sn.ID(),
				ShortID:  sn.ID().Str(),
				GroupKey: key,
			}
			snapshotres = append(snapshotres, k)
		}
	}
	return snapshotres, nil
}

// FilterLastestSnapshots filters a list of snapshots to only return
// the limit last entries for each hostname and path. If the snapshot
// contains multiple paths, they will be joined and treated as one
// item.
func FilterLastestSnapshots(list restic.Snapshots, limit int) restic.Snapshots {
	// Sort the snapshots so that the newer ones are listed first
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Time.After(list[j].Time)
	})

	var results restic.Snapshots
	seen := make(map[filterLastSnapshotsKey]int)
	for _, sn := range list {
		key := newFilterLastSnapshotsKey(sn)
		if seen[key] < limit {
			seen[key]++
			results = append(results, sn)
		}
	}
	return results
}

// newFilterLastSnapshotsKey initializes a filterLastSnapshotsKey from a Snapshot
func newFilterLastSnapshotsKey(sn *restic.Snapshot) filterLastSnapshotsKey {
	// Shallow slice copy
	var paths = make([]string, len(sn.Paths))
	copy(paths, sn.Paths)
	sort.Strings(paths)
	return filterLastSnapshotsKey{sn.Hostname, strings.Join(paths, "|")}
}
