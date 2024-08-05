package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/utils"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type lsSnapshot struct {
	*restic.Snapshot
	ID         *restic.ID `json:"id"`
	ShortID    string     `json:"short_id"`
	StructType string     `json:"struct_type"` // "snapshot"
}
type lsNode struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Path        string      `json:"path"`
	UID         uint32      `json:"uid"`
	GID         uint32      `json:"gid"`
	Size        string      `json:"size,omitempty"`
	Mode        os.FileMode `json:"mode,omitempty"`
	Permissions string      `json:"permissions,omitempty"`
	ModTime     time.Time   `json:"mtime,omitempty"`
	AccessTime  time.Time   `json:"atime,omitempty"`
	ChangeTime  time.Time   `json:"ctime,omitempty"`
	StructType  string      `json:"struct_type"` // "node"

	size uint64 // Target for Size pointer.
}
type LsRes struct {
	Snapshot lsSnapshot    `json:"snapshot"`
	Nodes    []interface{} `json:"nodes"`
}

func addNode(nodes []interface{}, path string, node *restic.Node) []interface{} {
	n := getNode(path, node)
	nodes = append(nodes, n)
	return nodes
}

func getNode(path string, node *restic.Node) interface{} {
	n := lsNode{
		Name:        node.Name,
		Type:        node.Type,
		Path:        path,
		UID:         node.UID,
		GID:         node.GID,
		size:        node.Size,
		Mode:        node.Mode,
		Permissions: node.Mode.String(),
		ModTime:     node.ModTime,
		AccessTime:  node.AccessTime,
		ChangeTime:  node.ChangeTime,
		StructType:  "node",
	}
	if node.Type == "file" {
		n.Size = utils.FormatBytes(n.size)
	}
	return n
}

func RunLs(targetP string, repoid int, snapshotid string) (*LsRes, error) {
	if snapshotid == "" {
		return nil, errors.Errorf("no snapshot ID specified")
	}
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

	snapshotLister, err := backend.MemorizeList(ctx, repo.Backend(), restic.SnapshotFile)
	if err != nil {
		return nil, err
	}

	sn, subfolder, err := (&restic.SnapshotFilter{}).FindLatest(ctx, snapshotLister, repo, snapshotid)
	if err != nil {
		return nil, err
	}

	sn.Tree, err = restic.FindTreeDirectory(ctx, repo, sn.Tree, subfolder)
	if err != nil {
		return nil, err
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
	tree, err := restic.LoadTree(ctx, repo, *sn.Tree)
	if err != nil {
		server.Logger().Error(err)
		return nil, fmt.Errorf("loadIndexing")
	}
	res, err := walk(ctx, repoHandler.repo, "/", tree, targetP)
	if err != nil {
		return nil, err
	}
	lsres.Nodes = res
	return &lsres, nil
}

// walk 列出 targetP 下所有dir或file，若targetP 为文件，则返回他自己
func walk(ctx context.Context, repo restic.BlobLoader, prefix string, tree *restic.Tree, targetP string) (res []interface{}, err error) {
	res = make([]interface{}, 0)
	sort.Slice(tree.Nodes, func(i, j int) bool {
		return tree.Nodes[i].Name < tree.Nodes[j].Name
	})

	for _, node := range tree.Nodes {
		p := path.Join(prefix, node.Name)
		if node.Type != "dir" {
			if p == targetP || strings.HasPrefix(p, targetP) {
				return addNode(res, p, node), nil
			}
			continue
		}
		if node.Subtree == nil {
			return res, errors.Errorf("subtree for node %v in tree %v is nil", node.Name, p)
		}
		if p == targetP {
			var dirnode []interface{}
			var filenode []interface{}
			subtree, err := restic.LoadTree(ctx, repo, *node.Subtree)
			if err != nil {
				return res, err
			}
			for _, subnode := range subtree.Nodes {
				subp := path.Join(p, subnode.Name)
				if subnode.Type == "dir" {
					dirnode = addNode(dirnode, subp, subnode)
				} else {
					filenode = addNode(filenode, subp, subnode)
				}
			}
			return append(dirnode, filenode...), nil
		}
		if fs.HasPathPrefix(p, targetP) {
			subtree, err := restic.LoadTree(ctx, repo, *node.Subtree)
			if err != nil {
				return res, err
			}
			return walk(ctx, repo, p, subtree, targetP)
		}
	}
	return res, nil
}
