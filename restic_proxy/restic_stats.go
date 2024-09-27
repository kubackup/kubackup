package resticProxy

import (
	"context"
	"fmt"
	"github.com/fanjindong/go-cache"
	"github.com/kubackup/kubackup/internal/consts"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/crypto"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/walker"
	"github.com/kubackup/kubackup/pkg/utils"
	"github.com/minio/sha256-simd"
	"gopkg.in/tomb.v2"
	"path/filepath"
	"sync"
	"time"
)

type StatsOptions struct {
	// the mode of counting to perform (see consts for available modes)
	countMode string

	restic.SnapshotFilter
}

var doing = false
var doinglock sync.Mutex

// GetAllRepoStats 获取所有仓库状态
func GetAllRepoStats() {
	doinglock.Lock()
	if doing {
		doinglock.Unlock()
		return
	}
	doing = true
	doinglock.Unlock()
	startime := time.Now()
	var t tomb.Tomb
	backupinfos := make([]model.BackupInfo, 0)
	maxDay := uint64(0)
	for _, repo := range Myrepositorys.rep {
		repoi := repo
		t.Go(func() error {
			snapshots, err := RunSnapshots(SnapshotOptions{}, repoi.repoId, make([]string, 0))
			if err != nil {
				server.Logger().Error(err)
				return err
			}
			daysec := uint64(0)
			if len(snapshots) > 0 {
				snres := snapshots[len(snapshots)-1].(SnapshotRes)
				daysec = uint64(time.Since(snres.Time) / time.Second)
				if daysec > maxDay {
					maxDay = daysec
				}
			}
			stats, err := runStats(StatsOptions{countMode: countModeUniqueFilesByContents}, repoi.repoId, []string{})
			if err != nil {
				server.Logger().Error(err)
				return err
			}
			stats2, err := runStats(StatsOptions{countMode: countModeRawData}, repoi.repoId, []string{})
			if err != nil {
				server.Logger().Error(err)
				return err
			}
			backupinfo := model.BackupInfo{
				RepositoryName:           repoi.repoName,
				FileTotal:                int(stats.TotalFileCount),
				DataDay:                  utils.FormatDay(daysec),
				DataSize:                 stats2.TotalSize,
				DataSizeStr:              utils.FormatBytes(stats2.TotalSize),
				SnapshotsNum:             stats.SnapshotsCount,
				CompressionSpaceSaving:   fmt.Sprintf("%.2f", stats2.CompressionSpaceSaving),
				TotalUncompressedSize:    stats2.TotalUncompressedSize,
				TotalUncompressedSizeStr: utils.FormatBytes(stats2.TotalUncompressedSize),
			}
			backupinfos = append(backupinfos, backupinfo)
			return nil
		})
	}
	if len(Myrepositorys.rep) > 0 {
		t.Kill(nil)
		_ = t.Wait()
	}
	filet := 0
	snapn := 0
	datas := uint64(0)
	uncompressed := uint64(0)

	for _, b := range backupinfos {
		filet = filet + b.FileTotal
		snapn = snapn + b.SnapshotsNum
		datas = datas + b.DataSize
		uncompressed = uncompressed + b.TotalUncompressedSize
	}
	duration := utils.FormatDuration(time.Since(startime))
	backupinfo := model.BackupInfo{
		FileTotal:                filet,
		DataDay:                  utils.FormatDay(maxDay),
		DataSize:                 datas,
		DataSizeStr:              utils.FormatBytes(datas),
		CompressionSpaceSaving:   fmt.Sprintf("%.2f", (1-float64(datas)/float64(uncompressed))*100),
		TotalUncompressedSize:    uncompressed,
		TotalUncompressedSizeStr: utils.FormatBytes(uncompressed),
		SnapshotsNum:             snapn,
		Time:                     time.Now(),
		Duration:                 duration,
	}
	key1 := consts.Key("GetAllRepoStats", "backupinfo")
	key2 := consts.Key("GetAllRepoStats", "backupinfos")
	c := server.Cache()
	c.Set(key1, backupinfo, cache.WithEx(24*time.Hour))
	c.Set(key2, backupinfos, cache.WithEx(24*time.Hour))
	doinglock.Lock()
	doing = false
	server.Logger().Info("结束执行GetAllRepoStats")
	doinglock.Unlock()
}

func runStats(opts StatsOptions, repoid int, snapshotIDs []string) (*StatsContainer, error) {
	err := verifyStatsInput(opts)
	if err != nil {
		return nil, err
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
	// create a container for the stats (and other needed state)
	stats := &StatsContainer{
		uniqueFiles:    make(map[fileID]struct{}),
		fileBlobs:      make(map[string]restic.IDSet),
		blobs:          restic.NewBlobSet(),
		SnapshotsCount: 0,
	}

	for sn := range FindFilteredSnapshots(ctx, snapshotLister, repo, &opts.SnapshotFilter, snapshotIDs) {
		err = statsWalkSnapshot(opts, ctx, sn, repo, stats)
		if err != nil {
			return nil, fmt.Errorf("error walking snapshot: %v", err)
		}
	}

	if opts.countMode == countModeRawData {
		// the blob handles have been collected, but not yet counted
		for blobHandle := range stats.blobs {
			pbs := repo.Index().Lookup(blobHandle)
			if len(pbs) == 0 {
				return nil, fmt.Errorf("blob %v not found", blobHandle)
			}
			stats.TotalSize += uint64(pbs[0].Length)
			if repo.Config().Version >= 2 {
				stats.TotalUncompressedSize += uint64(crypto.CiphertextLength(int(pbs[0].DataLength())))
				if pbs[0].IsCompressed() {
					stats.TotalCompressedBlobsSize += uint64(pbs[0].Length)
					stats.TotalCompressedBlobsUncompressedSize += uint64(crypto.CiphertextLength(int(pbs[0].DataLength())))
				}
			}
			stats.TotalBlobCount++
		}
		if stats.TotalCompressedBlobsSize > 0 {
			stats.CompressionRatio = float64(stats.TotalCompressedBlobsUncompressedSize) / float64(stats.TotalCompressedBlobsSize)
		}
		if stats.TotalUncompressedSize > 0 {
			stats.CompressionProgress = float64(stats.TotalCompressedBlobsUncompressedSize) / float64(stats.TotalUncompressedSize) * 100
			stats.CompressionSpaceSaving = (1 - float64(stats.TotalSize)/float64(stats.TotalUncompressedSize)) * 100
		}
	}
	return stats, nil
}

func statsWalkSnapshot(statsOptions StatsOptions, ctx context.Context, snapshot *restic.Snapshot, repo restic.Repository, stats *StatsContainer) error {
	if snapshot.Tree == nil {
		return fmt.Errorf("snapshot %s has nil tree", snapshot.ID().Str())
	}

	stats.SnapshotsCount++

	if statsOptions.countMode == countModeRawData {
		// count just the sizes of unique blobs; we don't need to walk the tree
		// ourselves in this case, since a nifty function does it for us
		return restic.FindUsedBlobs(ctx, repo, restic.IDs{*snapshot.Tree}, stats.blobs, nil)
	}

	uniqueInodes := make(map[uint64]struct{})
	err := walker.Walk(ctx, repo, *snapshot.Tree, restic.NewIDSet(), statsWalkTree(statsOptions, repo, stats, uniqueInodes))
	if err != nil {
		return fmt.Errorf("walking tree %s: %v", *snapshot.Tree, err)
	}

	return nil
}

func statsWalkTree(statsOptions StatsOptions, repo restic.Repository, stats *StatsContainer, uniqueInodes map[uint64]struct{}) walker.WalkFunc {
	return func(parentTreeID restic.ID, npath string, node *restic.Node, nodeErr error) (bool, error) {
		if nodeErr != nil {
			return true, nodeErr
		}
		if node == nil {
			return true, nil
		}

		if statsOptions.countMode == countModeUniqueFilesByContents || statsOptions.countMode == countModeBlobsPerFile {
			// only count this file if we haven't visited it before
			fid := makeFileIDByContents(node)
			if _, ok := stats.uniqueFiles[fid]; !ok {
				// mark the file as visited
				stats.uniqueFiles[fid] = struct{}{}

				if statsOptions.countMode == countModeUniqueFilesByContents {
					// simply count the size of each unique file (unique by contents only)
					stats.TotalSize += node.Size
					stats.TotalFileCount++
				}
				if statsOptions.countMode == countModeBlobsPerFile {
					// count the size of each unique blob reference, which is
					// by unique file (unique by contents and file path)
					for _, blobID := range node.Content {
						// ensure we have this file (by path) in our map; in this
						// mode, a file is unique by both contents and path
						nodePath := filepath.Join(npath, node.Name)
						if _, ok := stats.fileBlobs[nodePath]; !ok {
							stats.fileBlobs[nodePath] = restic.NewIDSet()
							stats.TotalFileCount++
						}
						if _, ok := stats.fileBlobs[nodePath][blobID]; !ok {
							// is always a data blob since we're accessing it via a file's Content array
							blobSize, found := repo.LookupBlobSize(blobID, restic.DataBlob)
							if !found {
								return true, fmt.Errorf("blob %s not found for tree %s", blobID, parentTreeID)
							}

							// count the blob's size, then add this blob by this
							// file (path) so we don't double-count it
							stats.TotalSize += uint64(blobSize)
							stats.fileBlobs[nodePath].Insert(blobID)
							// this mode also counts total unique blob _references_ per file
							stats.TotalBlobCount++
						}
					}
				}
			}
		}

		if statsOptions.countMode == countModeRestoreSize {
			// as this is a file in the snapshot, we can simply count its
			// size without worrying about uniqueness, since duplicate files
			// will still be restored
			stats.TotalFileCount++

			// if inodes are present, only count each inode once
			// (hard links do not increase restore size)
			if _, ok := uniqueInodes[node.Inode]; !ok || node.Inode == 0 {
				uniqueInodes[node.Inode] = struct{}{}
				stats.TotalSize += node.Size
			}

			return false, nil
		}

		return true, nil
	}
}

// makeFileIDByContents returns a hash of the blob IDs of the
// node's Content in sequence.
func makeFileIDByContents(node *restic.Node) fileID {
	var bb []byte
	for _, c := range node.Content {
		bb = append(bb, []byte(c[:])...)
	}
	return sha256.Sum256(bb)
}

func verifyStatsInput(options StatsOptions) error {
	// require a recognized counting mode
	switch options.countMode {
	case countModeRestoreSize:
	case countModeUniqueFilesByContents:
	case countModeBlobsPerFile:
	case countModeRawData:
	default:
		return fmt.Errorf("unknown counting mode: %s (counting mode: restore-size (default), files-by-contents, blobs-per-file or raw-data)", options.countMode)
	}

	return nil
}

// statsContainer holds information during a walk of a repository
// to collect information about it, as well as state needed
// for a successful and efficient walk.
type StatsContainer struct {
	TotalSize                            uint64  `json:"total_size"`
	TotalUncompressedSize                uint64  `json:"total_uncompressed_size,omitempty"`
	TotalCompressedBlobsSize             uint64  `json:"-"`
	TotalCompressedBlobsUncompressedSize uint64  `json:"-"`
	CompressionRatio                     float64 `json:"compression_ratio,omitempty"`
	CompressionProgress                  float64 `json:"compression_progress,omitempty"`
	CompressionSpaceSaving               float64 `json:"compression_space_saving,omitempty"`
	TotalFileCount                       uint64  `json:"total_file_count,omitempty"`
	TotalBlobCount                       uint64  `json:"total_blob_count,omitempty"`
	// holds count of all considered snapshots
	SnapshotsCount int `json:"snapshots_count"`

	// uniqueFiles marks visited files according to their
	// contents (hashed sequence of content blob IDs)
	uniqueFiles map[fileID]struct{}

	// fileBlobs maps a file name (path) to the set of
	// blobs that have been seen as a part of the file
	fileBlobs map[string]restic.IDSet

	// blobs is used to count individual unique blobs,
	// independent of references to files
	blobs restic.BlobSet
}

// fileID is a 256-bit hash that distinguishes unique files.
type fileID [32]byte

const (
	countModeRestoreSize           = "restore-size"
	countModeUniqueFilesByContents = "files-by-contents"
	countModeBlobsPerFile          = "blobs-per-file"
	countModeRawData               = "raw-data"
)
