package resticProxy

import (
	"context"
	"fmt"
	operationModel "github.com/kubackup/kubackup/internal/entity/v1/operation"
	repoModel "github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/log"
	"github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/utils"
	"gopkg.in/tomb.v2"
	"math"
	"sort"
	"strconv"
	"strings"
)

var errorIndexIncomplete = errors.Fatal("index is not complete")
var errorPacksMissing = errors.Fatal("packs from index missing in repo")
var errorSizeNotMatching = errors.Fatal("pack size does not match calculated size from index")

// PruneOptions collects all options for the cleanup command.
type PruneOptions struct {
	MaxUnused      string
	maxUnusedBytes func(used uint64) (unused uint64) // calculates the number of unused bytes after repacking, according to MaxUnused

	MaxRepackSize  string
	MaxRepackBytes uint64

	RepackCachableOnly bool
}

func verifyPruneOptions(opts *PruneOptions) error {
	if len(opts.MaxRepackSize) > 0 {
		size, err := parseSizeStr(opts.MaxRepackSize)
		if err != nil {
			return err
		}
		opts.MaxRepackBytes = uint64(size)
	}

	maxUnused := strings.TrimSpace(opts.MaxUnused)
	if maxUnused == "" {
		return errors.Fatalf("invalid value for --max-unused: %q", opts.MaxUnused)
	}

	// parse MaxUnused either as unlimited, a percentage, or an absolute number of bytes
	switch {
	case maxUnused == "unlimited":
		opts.maxUnusedBytes = func(used uint64) uint64 {
			return math.MaxUint64
		}

	case strings.HasSuffix(maxUnused, "%"):
		maxUnused = strings.TrimSuffix(maxUnused, "%")
		p, err := strconv.ParseFloat(maxUnused, 64)
		if err != nil {
			return errors.Fatalf("invalid percentage %q passed for --max-unused: %v", opts.MaxUnused, err)
		}

		if p < 0 {
			return errors.Fatal("percentage for --max-unused must be positive")
		}

		if p >= 100 {
			return errors.Fatal("percentage for --max-unused must be below 100%")
		}

		opts.maxUnusedBytes = func(used uint64) uint64 {
			return uint64(p / (100 - p) * float64(used))
		}

	default:
		size, err := parseSizeStr(maxUnused)
		if err != nil {
			return errors.Fatalf("invalid number of bytes %q for --max-unused: %v", opts.MaxUnused, err)
		}

		opts.maxUnusedBytes = func(used uint64) uint64 {
			return uint64(size)
		}
	}

	return nil
}

func RunPrune(opts PruneOptions, repoid int) (int, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return 0, err
	}
	repo := repoHandler.repo

	err = verifyPruneOptions(&opts)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithCancel(repoHandler.gopts.ctx)
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	lock, err := lockRepoExclusive(ctx, repo)
	if err != nil {
		clean.Cleanup()
		return 0, err
	}
	clean.AddCleanCtx(func() {
		unlockRepo(lock)
	})
	status := repoModel.StatusNone
	oper := operationModel.Operation{
		RepositoryId: repoid,
		Type:         operationModel.PRUNE_TYPE,
		Status:       status,
		Logs:         make([]*wsTaskInfo.Sprint, 0),
	}
	err = operationService.Create(&oper, common.DBOptions{})
	if err != nil {
		clean.Cleanup()
		return 0, err
	}
	var t tomb.Tomb
	logTask := log.LogInfo{}
	logTask.SetId(oper.Id)
	spr := wsTaskInfo.NewSprintf(&logTask)

	logTask.SetBound(make(chan error))
	log.LogInfos.Set(oper.Id, &logTask)
	t.Go(func() error {
		for {
			select {
			case <-t.Context(ctx).Done():
				return nil
			case <-log.LogInfos.Get(oper.Id).GetBound():
				info := log.LogInfos.Get(oper.Id)
				spr.UpdateTaskInfo(info)
				spr.SendAllLog()
			}
		}
	})

	t.Go(func() error {
		defer clean.Cleanup()
		err := runPruneWithRepo(opts, ctx, repo, restic.NewIDSet(), spr)
		status = repoModel.StatusNone
		if err != nil {
			spr.Append(wsTaskInfo.Error, err.Error())
			status = repoModel.StatusErr
		} else {
			status = repoModel.StatusRun
		}
		err = LoadIndex(ctx, repo)
		if err != nil {
			spr.Append(wsTaskInfo.Error, err.Error())
		}
		oper.Status = status
		oper.Logs = spr.Sprints
		err = operationService.Update(&oper, common.DBOptions{})
		if err != nil {
			server.Logger().Error(err)
		}
		t.Kill(nil)
		log.LogInfos.Close(oper.Id, "process end", 1)
		return nil
	})
	return oper.Id, nil
}

func runPruneWithRepo(opts PruneOptions, ctx context.Context, repo *repository.Repository, ignoreSnapshots restic.IDSet, spr *wsTaskInfo.Sprintf) error {
	// we do not need index updates while pruning!
	//repo.DisableAutoIndexUpdate()

	if repo.Cache == nil {
		spr.Append(wsTaskInfo.Warning, "running prune without a cache, this may be very slow!\n")
	}
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("loading indexes...\n"))
	//err := LoadIndex(gopts.ctx, repo)
	//if err != nil {
	//	return err
	//}

	usedBlobs, err := getUsedBlobs(ctx, repo, ignoreSnapshots, spr)
	if err != nil {
		spr.Append(wsTaskInfo.Error, err.Error())
		return err
	}
	return prune(opts, ctx, repo, usedBlobs, spr)
}

type packInfo struct {
	usedBlobs      uint
	unusedBlobs    uint
	duplicateBlobs uint
	usedSize       uint64
	unusedSize     uint64
	tpe            restic.BlobType
}

type packInfoWithID struct {
	ID restic.ID
	packInfo
}

// prune selects which files to rewrite and then does that. The map usedBlobs is
// modified in the process.
func prune(opts PruneOptions, ctx context.Context, repo restic.Repository, usedBlobs restic.BlobSet, spr *wsTaskInfo.Sprintf) error {

	var stats struct {
		blobs struct {
			used      uint
			duplicate uint
			unused    uint
			remove    uint
			repack    uint
			repackrm  uint
		}
		size struct {
			used      uint64
			duplicate uint64
			unused    uint64
			remove    uint64
			repack    uint64
			repackrm  uint64
			unref     uint64
		}
		packs struct {
			used       uint
			unused     uint
			partlyUsed uint
			keep       uint
		}
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("searching used packs...\n"))

	keepBlobs := restic.NewBlobSet()
	duplicateBlobs := restic.NewBlobSet()

	// iterate over all blobs in index to find out which blobs are duplicates
	for blob := range repo.Index().Each(ctx) {
		bh := blob.BlobHandle
		size := uint64(blob.Length)
		switch {
		case usedBlobs.Has(bh): // used blob, move to keepBlobs
			usedBlobs.Delete(bh)
			keepBlobs.Insert(bh)
			stats.size.used += size
			stats.blobs.used++
		case keepBlobs.Has(bh): // duplicate blob
			duplicateBlobs.Insert(bh)
			stats.size.duplicate += size
			stats.blobs.duplicate++
		default:
			stats.size.unused += size
			stats.blobs.unused++
		}
	}

	// Check if all used blobs have been found in index
	if len(usedBlobs) != 0 {
		spr.Append(wsTaskInfo.Warning, fmt.Sprintf("%v not found in the index\n\n"+
			"Integrity check failed: Data seems to be missing.\n"+
			"Will not start prune to prevent (additional) data loss!\n"+
			"Please report this error (along with the output of the 'prune' run) at\n"+
			"https://github.com/restic/restic/issues/new/choose\n", usedBlobs))
		spr.Append(wsTaskInfo.Error, errorIndexIncomplete.Error())
		return errorIndexIncomplete
	}

	indexPack := make(map[restic.ID]packInfo)

	// save computed pack header size
	for pid, hdrSize := range repo.Index().PackSize(ctx, true) {
		// initialize tpe with NumBlobTypes to indicate it's not set
		indexPack[pid] = packInfo{tpe: restic.NumBlobTypes, usedSize: uint64(hdrSize)}
	}

	// iterate over all blobs in index to generate packInfo
	for blob := range repo.Index().Each(ctx) {
		ip := indexPack[blob.PackID]

		// Set blob type if not yet set
		if ip.tpe == restic.NumBlobTypes {
			ip.tpe = blob.Type
		}

		// mark mixed packs with "Invalid blob type"
		if ip.tpe != blob.Type {
			ip.tpe = restic.InvalidBlob
		}

		bh := blob.BlobHandle
		size := uint64(blob.Length)
		switch {
		case duplicateBlobs.Has(bh): // duplicate blob
			ip.usedSize += size
			ip.duplicateBlobs++
		case keepBlobs.Has(bh): // used blob, not duplicate
			ip.usedSize += size
			ip.usedBlobs++
		default: // unused blob
			ip.unusedSize += size
			ip.unusedBlobs++
		}
		// update indexPack
		indexPack[blob.PackID] = ip
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("collecting packs for deletion and repacking\n"))
	removePacksFirst := restic.NewIDSet()
	removePacks := restic.NewIDSet()
	repackPacks := restic.NewIDSet()

	var repackCandidates []packInfoWithID
	repackAllPacksWithDuplicates := true

	keep := func(p packInfo) {
		stats.packs.keep++
		if p.duplicateBlobs > 0 {
			repackAllPacksWithDuplicates = false
		}
	}

	// loop over all packs and decide what to do
	pro := newProgressMax(true, uint64(len(indexPack)), "packs processed", spr)
	spr.ResetLimitNum()
	err := repo.List(ctx, restic.PackFile, func(id restic.ID, packSize int64) error {
		p, ok := indexPack[id]
		if !ok {
			// Pack was not referenced in index and is not used  => immediately remove!
			spr.AppendLimit(wsTaskInfo.Info, fmt.Sprintf("will remove pack %v as it is unused and not indexed\n", id.Str()))
			removePacksFirst.Insert(id)
			stats.size.unref += uint64(packSize)
			return nil
		}

		if p.unusedSize+p.usedSize != uint64(packSize) &&
			!(p.usedBlobs == 0 && p.duplicateBlobs == 0) {
			// Pack size does not fit and pack is needed => error
			// If the pack is not needed, this is no error, the pack can
			// and will be simply removed, see below.
			spr.AppendLimit(wsTaskInfo.Info, fmt.Sprintf("warning: pack %s: calculated size %d does not match real size %d\nRun 'restic rebuild-index'.\n",
				id.Str(), p.unusedSize+p.usedSize, packSize))
			spr.Append(wsTaskInfo.Error, errorSizeNotMatching.Error())
			return errorSizeNotMatching
		}

		// statistics
		switch {
		case p.usedBlobs == 0 && p.duplicateBlobs == 0:
			stats.packs.unused++
		case p.unusedBlobs == 0:
			stats.packs.used++
		default:
			stats.packs.partlyUsed++
		}

		// decide what to do
		switch {
		case p.usedBlobs == 0 && p.duplicateBlobs == 0:
			// All blobs in pack are no longer used => remove pack!
			removePacks.Insert(id)
			stats.blobs.remove += p.unusedBlobs
			stats.size.remove += p.unusedSize

		case opts.RepackCachableOnly && p.tpe == restic.DataBlob:
			// if this is a data pack and --repack-cacheable-only is set => keep pack!
			keep(p)

		case p.unusedBlobs == 0 && p.duplicateBlobs == 0 && p.tpe != restic.InvalidBlob:
			// All blobs in pack are used and not duplicates/mixed => keep pack!
			keep(p)

		default:
			// all other packs are candidates for repacking
			repackCandidates = append(repackCandidates, packInfoWithID{ID: id, packInfo: p})
		}

		delete(indexPack, id)
		pro.Add(1)
		return nil
	})
	pro.Done()
	if err != nil {
		return err
	}

	// At this point indexPacks contains only missing packs!

	// missing packs that are not needed can be ignored
	ignorePacks := restic.NewIDSet()
	for id, p := range indexPack {
		if p.usedBlobs == 0 && p.duplicateBlobs == 0 {
			ignorePacks.Insert(id)
			stats.blobs.remove += p.unusedBlobs
			stats.size.remove += p.unusedSize
			delete(indexPack, id)
		}
	}

	if len(indexPack) != 0 {
		spr.Append(wsTaskInfo.Warning, fmt.Sprintf("The index references %d needed pack files which are missing from the repository:\n", len(indexPack)))
		spr.ResetLimitNum()
		for id := range indexPack {
			spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("  %v\n", id))
		}
		spr.Append(wsTaskInfo.Error, errorPacksMissing.Error())
		return errorPacksMissing
	}
	if len(ignorePacks) != 0 {
		spr.Append(wsTaskInfo.Warning, fmt.Sprintf("Missing but unneeded pack files are referenced in the index, will be repaired\n"))
		spr.ResetLimitNum()
		for id := range ignorePacks {
			spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("will forget missing pack file %v\n", id))
		}
	}

	// calculate limit for number of unused bytes in the repo after repacking
	maxUnusedSizeAfter := opts.maxUnusedBytes(stats.size.used)

	// Sort repackCandidates such that packs with highest ratio unused/used space are picked first.
	// This is equivalent to sorting by unused / total space.
	// Instead of unused[i] / used[i] > unused[j] / used[j] we use
	// unused[i] * used[j] > unused[j] * used[i] as uint32*uint32 < uint64
	// Morover duplicates and packs containing trees are sorted to the beginning
	sort.Slice(repackCandidates, func(i, j int) bool {
		pi := repackCandidates[i].packInfo
		pj := repackCandidates[j].packInfo
		switch {
		case pi.duplicateBlobs > 0 && pj.duplicateBlobs == 0:
			return true
		case pj.duplicateBlobs > 0 && pi.duplicateBlobs == 0:
			return false
		case pi.tpe != restic.DataBlob && pj.tpe == restic.DataBlob:
			return true
		case pj.tpe != restic.DataBlob && pi.tpe == restic.DataBlob:
			return false
		}
		return pi.unusedSize*pj.usedSize > pj.unusedSize*pi.usedSize
	})

	repack := func(id restic.ID, p packInfo) {
		repackPacks.Insert(id)
		stats.blobs.repack += p.unusedBlobs + p.duplicateBlobs + p.usedBlobs
		stats.size.repack += p.unusedSize + p.usedSize
		stats.blobs.repackrm += p.unusedBlobs
		stats.size.repackrm += p.unusedSize
	}

	for _, p := range repackCandidates {
		reachedUnusedSizeAfter := (stats.size.unused-stats.size.remove-stats.size.repackrm < maxUnusedSizeAfter)

		reachedRepackSize := false
		if opts.MaxRepackBytes > 0 {
			reachedRepackSize = stats.size.repack+p.unusedSize+p.usedSize > opts.MaxRepackBytes
		}

		switch {
		case reachedRepackSize:
			keep(p.packInfo)

		case p.duplicateBlobs > 0, p.tpe != restic.DataBlob:
			// repacking duplicates/non-data is only limited by repackSize
			repack(p.ID, p.packInfo)

		case reachedUnusedSizeAfter:
			// for all other packs stop repacking if tolerated unused size is reached.
			keep(p.packInfo)

		default:
			repack(p.ID, p.packInfo)
		}
	}

	// if all duplicates are repacked, print out correct statistics
	if repackAllPacksWithDuplicates {
		stats.blobs.repackrm += stats.blobs.duplicate
		stats.size.repackrm += stats.size.duplicate
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("used:        %10d blobs / %s\n", stats.blobs.used, utils.FormatBytes(stats.size.used)))
	if stats.blobs.duplicate > 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("duplicates:  %10d blobs / %s\n", stats.blobs.duplicate, utils.FormatBytes(stats.size.duplicate)))
	}
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused:      %10d blobs / %s\n", stats.blobs.unused, utils.FormatBytes(stats.size.unused)))
	if stats.size.unref > 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("unreferenced:                   %s\n", utils.FormatBytes(stats.size.unref)))
	}
	totalBlobs := stats.blobs.used + stats.blobs.unused + stats.blobs.duplicate
	totalSize := stats.size.used + stats.size.duplicate + stats.size.unused + stats.size.unref
	unusedSize := stats.size.duplicate + stats.size.unused
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("total:       %10d blobs / %s\n", totalBlobs, utils.FormatBytes(totalSize)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused size: %s of total size\n", utils.FormatPercent(unusedSize, totalSize)))

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to repack:   %10d blobs / %s\n", stats.blobs.repack, utils.FormatBytes(stats.size.repack)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("this removes %10d blobs / %s\n", stats.blobs.repackrm, utils.FormatBytes(stats.size.repackrm)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to delete:   %10d blobs / %s\n", stats.blobs.remove, utils.FormatBytes(stats.size.remove+stats.size.unref)))
	totalPruneSize := stats.size.remove + stats.size.repackrm + stats.size.unref
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("total prune: %10d blobs / %s\n", stats.blobs.remove+stats.blobs.repackrm, utils.FormatBytes(totalPruneSize)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("remaining:   %10d blobs / %s\n", totalBlobs-(stats.blobs.remove+stats.blobs.repackrm), utils.FormatBytes(totalSize-totalPruneSize)))
	unusedAfter := unusedSize - stats.size.remove - stats.size.repackrm
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused size after prune: %s (%s of remaining size)\n",
		utils.FormatBytes(unusedAfter), utils.FormatPercent(unusedAfter, totalSize-totalPruneSize)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("totally used packs: %10d\n", stats.packs.used))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("partly used packs:  %10d\n", stats.packs.partlyUsed))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused packs:       %10d\n\n", stats.packs.unused))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to keep:   %10d packs\n", stats.packs.keep))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to repack: %10d packs\n", len(repackPacks)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to delete: %10d packs\n", len(removePacks)))
	if len(removePacksFirst) > 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("to delete: %10d unreferenced packs\n\n", len(removePacksFirst)))
	}

	// unreferenced packs can be safely deleted first
	if len(removePacksFirst) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("deleting unreferenced packs\n"))
		DeleteFiles(spr, ctx, repo, removePacksFirst, restic.PackFile)
	}

	if len(repackPacks) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("repacking packs\n"))
		max := uint64(len(repackPacks))
		pro := newProgressMax(true, max, "packs repacked", spr)
		_, err := repository.Repack(ctx, repo, repackPacks, keepBlobs, pro)
		pro.Done()
		if err != nil {
			return errors.Fatalf("%s", err)
		}

		// Also remove repacked packs
		removePacks.Merge(repackPacks)
	}

	if len(ignorePacks) == 0 {
		ignorePacks = removePacks
	} else {
		ignorePacks.Merge(removePacks)
	}

	if len(ignorePacks) != 0 {
		err = rebuildIndexFiles(ctx, repo, ignorePacks, nil, spr)
		if err != nil {
			return errors.Fatalf("%s", err)
		}
	}

	if len(removePacks) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("removing %d old packs\n", len(removePacks)))
		DeleteFiles(spr, ctx, repo, removePacks, restic.PackFile)
	}
	spr.Append(wsTaskInfo.Success, fmt.Sprintf("done\n"))
	return nil
}

func rebuildIndexFiles(ctx context.Context, repo restic.Repository, removePacks restic.IDSet, extraObsolete restic.IDs, spr *wsTaskInfo.Sprintf) error {
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("rebuilding index\n"))

	idx := (repo.Index()).(*repository.MasterIndex)
	packcount := uint64(len(idx.Packs(removePacks)))
	pro := newProgressMax(true, packcount, "packs processed", spr)
	obsoleteIndexes, err := idx.Save(ctx, repo, removePacks, extraObsolete, pro)
	pro.Done()
	if err != nil {
		return err
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("deleting obsolete index files\n"))
	return DeleteFilesChecked(spr, ctx, repo, obsoleteIndexes, restic.IndexFile)
}

func getUsedBlobs(ctx context.Context, repo restic.Repository, ignoreSnapshots restic.IDSet, spr *wsTaskInfo.Sprintf) (usedBlobs restic.BlobSet, err error) {

	var snapshotTrees restic.IDs
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("loading all snapshots...\n"))
	err = restic.ForAllSnapshots(ctx, repo, ignoreSnapshots,
		func(id restic.ID, sn *restic.Snapshot, err error) error {
			server.Logger().Debugf("add snapshot %v (tree %v, error %v)", id, *sn.Tree, err)
			if err != nil {
				return err
			}
			snapshotTrees = append(snapshotTrees, *sn.Tree)
			return nil
		})
	if err != nil {
		return nil, err
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("finding data that is still in use for %d snapshots\n", len(snapshotTrees)))

	usedBlobs = restic.NewBlobSet()

	pro := newProgressMax(true, uint64(len(snapshotTrees)), "snapshots", spr)
	err = restic.FindUsedBlobs(ctx, repo, snapshotTrees, usedBlobs, pro)
	pro.Done()
	if err != nil {
		if repo.Backend().IsNotExist(err) {
			return nil, errors.Fatal("unable to load a tree from the repo: " + err.Error())
		}

		return nil, err
	}
	return usedBlobs, nil
}
