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
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/index"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/pack"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui"
	"github.com/kubackup/kubackup/pkg/utils"
	"gopkg.in/tomb.v2"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var errorIndexIncomplete = errors.Fatal("index is not complete")
var errorPacksMissing = errors.Fatal("packs from index missing in repo")
var errorSizeNotMatching = errors.Fatal("pack size does not match calculated size from index")

// PruneOptions collects all options for the cleanup command.
type PruneOptions struct {
	DryRun                bool
	UnsafeNoSpaceRecovery string

	unsafeRecovery bool

	MaxUnused      string
	maxUnusedBytes func(used uint64) (unused uint64) // calculates the number of unused bytes after repacking, according to MaxUnused

	MaxRepackSize  string
	MaxRepackBytes uint64

	RepackCachableOnly bool
	RepackSmall        bool
	RepackUncompressed bool
}

type pruneStats struct {
	blobs struct {
		used      uint
		duplicate uint
		unused    uint
		remove    uint
		repack    uint
		repackrm  uint
	}
	size struct {
		used         uint64
		duplicate    uint64
		unused       uint64
		remove       uint64
		repack       uint64
		repackrm     uint64
		unref        uint64
		uncompressed uint64
	}
	packs struct {
		used       uint
		unused     uint
		partlyUsed uint
		unref      uint
		keep       uint
		repack     uint
		remove     uint
	}
}

type prunePlan struct {
	removePacksFirst restic.IDSet          // packs to remove first (unreferenced packs)
	repackPacks      restic.IDSet          // packs to repack
	keepBlobs        restic.CountedBlobSet // blobs to keep during repacking
	removePacks      restic.IDSet          // packs to remove
	ignorePacks      restic.IDSet          // packs to ignore when rebuilding the index
}

func verifyPruneOptions(opts *PruneOptions) error {
	opts.MaxRepackBytes = math.MaxUint64
	if len(opts.MaxRepackSize) > 0 {
		size, err := ui.ParseBytes(opts.MaxRepackSize)
		if err != nil {
			return err
		}
		opts.MaxRepackBytes = uint64(size)
	}
	if opts.UnsafeNoSpaceRecovery != "" {
		// prevent repacking data to make sure users cannot get stuck.
		opts.MaxRepackBytes = 0
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
		size, err := ui.ParseBytes(maxUnused)
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
	if repoHandler.gopts.Compression != 1 {
		opts.RepackUncompressed = true
	}
	if err != nil {
		return 0, err
	}
	repo := repoHandler.repo

	if repo.Backend().Connections() < 2 {
		return 0, errors.Fatal("prune requires a backend connection limit of at least two")
	}

	if repo.Config().Version < 2 && opts.RepackUncompressed {
		return 0, errors.Fatal("compression requires at least repository format version 2")
	}

	err = verifyPruneOptions(&opts)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithCancel(context.Background())
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

	logTask.SetBound(make(chan string))
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
		err = repo.LoadIndex(ctx, nil)
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
	repo.DisableAutoIndexUpdate()

	if repo.Cache == nil {
		spr.Append(wsTaskInfo.Warning, "running prune without a cache, this may be very slow!\n")
	}

	plan, stats, err := planPrune(ctx, opts, repo, ignoreSnapshots, spr)
	if err != nil {
		spr.Append(wsTaskInfo.Error, err.Error())
		return err
	}
	err = printPruneStats(stats, spr)
	if err != nil {
		return err
	}
	// Trigger GC to reset garbage collection threshold
	runtime.GC()

	return prune(opts, ctx, repo, plan, spr)
}

type packInfo struct {
	usedBlobs    uint
	unusedBlobs  uint
	usedSize     uint64
	unusedSize   uint64
	tpe          restic.BlobType
	uncompressed bool
}

type packInfoWithID struct {
	ID restic.ID
	packInfo
	mustCompress bool
}

// prune selects which files to rewrite and then does that. The map usedBlobs is
// modified in the process.
func prune(opts PruneOptions, ctx context.Context, repo restic.Repository, plan prunePlan, spr *wsTaskInfo.Sprintf) error {

	// unreferenced packs can be safely deleted first
	if len(plan.removePacksFirst) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("deleting unreferenced packs\n"))
		DeleteFiles(spr, ctx, repo, plan.removePacksFirst, restic.PackFile)
	}

	if len(plan.repackPacks) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("repacking packs\n"))
		max := uint64(len(plan.repackPacks))
		pro := newProgressMax(true, max, "packs repacked", spr)
		_, err := repository.Repack(ctx, repo, repo, plan.repackPacks, plan.keepBlobs, pro)
		pro.Done()
		if err != nil {
			return errors.Fatalf("%s", err)
		}

		// Also remove repacked packs
		plan.removePacks.Merge(plan.repackPacks)
		if len(plan.keepBlobs) != 0 {
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("%v was not repacked\n\n"+
				"Integrity check failed.\n"+
				"Please report this error (along with the output of the 'prune' run) at\n"+
				"https://github.com/restic/restic/issues/new/choose\n", plan.keepBlobs))
			return errors.Fatal("internal error: blobs were not repacked")
		}

		// allow GC of the blob set
		plan.keepBlobs = nil
	}

	if len(plan.ignorePacks) == 0 {
		plan.ignorePacks = plan.removePacks
	} else {
		plan.ignorePacks.Merge(plan.removePacks)
	}
	if opts.unsafeRecovery {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("deleting index files\n"))
		indexFiles := repo.Index().(*index.MasterIndex).IDs()
		err := DeleteFilesChecked(spr, ctx, repo, indexFiles, restic.IndexFile)
		if err != nil {
			return errors.Fatalf("%s", err)
		}
	} else if len(plan.ignorePacks) != 0 {
		err := rebuildIndexFiles(ctx, repo, plan.ignorePacks, nil, spr)
		if err != nil {
			return errors.Fatalf("%s", err)
		}
	}

	if len(plan.removePacks) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("removing %d old packs\n", len(plan.removePacks)))
		DeleteFiles(spr, ctx, repo, plan.removePacks, restic.PackFile)
	}

	if opts.unsafeRecovery {
		_, err := writeIndexFiles(ctx, repo, plan.ignorePacks, nil, spr)
		if err != nil {
			return errors.Fatalf("%s", err)
		}
	}
	spr.Append(wsTaskInfo.Success, fmt.Sprintf("done\n"))
	return nil
}

func writeIndexFiles(ctx context.Context, repo restic.Repository, removePacks restic.IDSet, extraObsolete restic.IDs, spr *wsTaskInfo.Sprintf) (restic.IDSet, error) {
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("rebuilding index\n"))
	bar := newProgressMax(true, 0, "packs processed", spr)
	obsoleteIndexes, err := repo.Index().Save(ctx, repo, removePacks, extraObsolete, bar)
	bar.Done()
	return obsoleteIndexes, err
}

func rebuildIndexFiles(ctx context.Context, repo restic.Repository, removePacks restic.IDSet, extraObsolete restic.IDs, spr *wsTaskInfo.Sprintf) error {

	obsoleteIndexes, err := writeIndexFiles(ctx, repo, removePacks, extraObsolete, spr)
	if err != nil {
		return err
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("deleting obsolete index files\n"))
	DeleteFiles(spr, ctx, repo, obsoleteIndexes, restic.IndexFile)
	return nil
}

// planPrune selects which files to rewrite and which to delete and which blobs to keep.
// Also some summary statistics are returned.
func planPrune(ctx context.Context, opts PruneOptions, repo restic.Repository, ignoreSnapshots restic.IDSet, spr *wsTaskInfo.Sprintf) (prunePlan, pruneStats, error) {
	var stats pruneStats

	usedBlobs, err := getUsedBlobs(ctx, repo, ignoreSnapshots, spr)
	if err != nil {
		return prunePlan{}, stats, err
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprint("searching used packs...\n"))
	keepBlobs, indexPack, err := packInfoFromIndex(ctx, repo.Index(), usedBlobs, &stats, spr)
	if err != nil {
		return prunePlan{}, stats, err
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprint("collecting packs for deletion and repacking\n"))
	plan, err := decidePackAction(ctx, opts, repo, indexPack, &stats, spr)
	if err != nil {
		return prunePlan{}, stats, err
	}

	if len(plan.repackPacks) != 0 {
		blobCount := keepBlobs.Len()
		// when repacking, we do not want to keep blobs which are
		// already contained in kept packs, so delete them from keepBlobs
		repo.Index().Each(ctx, func(blob restic.PackedBlob) {
			if plan.removePacks.Has(blob.PackID) || plan.repackPacks.Has(blob.PackID) {
				return
			}
			keepBlobs.Delete(blob.BlobHandle)
		})

		if keepBlobs.Len() < blobCount/2 {
			// replace with copy to shrink map to necessary size if there's a chance to benefit
			keepBlobs = keepBlobs.Copy()
		}
	} else {
		// keepBlobs is only needed if packs are repacked
		keepBlobs = nil
	}
	plan.keepBlobs = keepBlobs

	return plan, stats, nil
}

// printPruneStats prints out the statistics
func printPruneStats(stats pruneStats, spr *wsTaskInfo.Sprintf) error {

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
	if stats.size.uncompressed > 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("not yet compressed:              %s\n", ui.FormatBytes(stats.size.uncompressed)))
	}
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("remaining:   %10d blobs / %s\n", totalBlobs-(stats.blobs.remove+stats.blobs.repackrm), utils.FormatBytes(totalSize-totalPruneSize)))
	unusedAfter := unusedSize - stats.size.remove - stats.size.repackrm
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused size after prune: %s (%s of remaining size)\n",
		utils.FormatBytes(unusedAfter), utils.FormatPercent(unusedAfter, totalSize-totalPruneSize)))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("totally used packs: %10d\n", stats.packs.used))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("partly used packs:  %10d\n", stats.packs.partlyUsed))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused packs:       %10d\n\n", stats.packs.unused))

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to keep:   %10d packs\n", stats.packs.keep))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to repack: %10d packs\n", stats.packs.repack))
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("to delete: %10d packs\n", stats.packs.remove))
	if stats.packs.unref > 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("to delete: %10d unreferenced packs\n\n", stats.packs.unref))
	}
	return nil
}

func packInfoFromIndex(ctx context.Context, idx restic.MasterIndex, usedBlobs restic.CountedBlobSet, stats *pruneStats, spr *wsTaskInfo.Sprintf) (restic.CountedBlobSet, map[restic.ID]packInfo, error) {
	// iterate over all blobs in index to find out which blobs are duplicates
	// The counter in usedBlobs describes how many instances of the blob exist in the repository index
	// Thus 0 == blob is missing, 1 == blob exists once, >= 2 == duplicates exist
	idx.Each(ctx, func(blob restic.PackedBlob) {
		bh := blob.BlobHandle
		count, ok := usedBlobs[bh]
		if ok {
			if count < math.MaxUint8 {
				// don't overflow, but saturate count at 255
				// this can lead to a non-optimal pack selection, but won't cause
				// problems otherwise
				count++
			}

			usedBlobs[bh] = count
		}
	})

	// Check if all used blobs have been found in index
	missingBlobs := restic.NewBlobSet()
	for bh, count := range usedBlobs {
		if count == 0 {
			// blob does not exist in any pack files
			missingBlobs.Insert(bh)
		}
	}

	if len(missingBlobs) != 0 {
		spr.Append(wsTaskInfo.Error, fmt.Sprintf("%v not found in the index\n\n"+
			"Integrity check failed: Data seems to be missing.\n"+
			"Will not start prune to prevent (additional) data loss!\n"+
			"Please report this error (along with the output of the 'prune' run) at\n"+
			"https://github.com/restic/restic/issues/new/choose\n", missingBlobs))
		return nil, nil, errorIndexIncomplete
	}

	indexPack := make(map[restic.ID]packInfo)

	// save computed pack header size
	for pid, hdrSize := range pack.Size(ctx, idx, true) {
		// initialize tpe with NumBlobTypes to indicate it's not set
		indexPack[pid] = packInfo{tpe: restic.NumBlobTypes, usedSize: uint64(hdrSize)}
	}

	hasDuplicates := false
	// iterate over all blobs in index to generate packInfo
	idx.Each(ctx, func(blob restic.PackedBlob) {
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
		dupCount := usedBlobs[bh]
		switch {
		case dupCount >= 2:
			hasDuplicates = true
			// mark as unused for now, we will later on select one copy
			ip.unusedSize += size
			ip.unusedBlobs++

			// count as duplicate, will later on change one copy to be counted as used
			stats.size.duplicate += size
			stats.blobs.duplicate++
		case dupCount == 1: // used blob, not duplicate
			ip.usedSize += size
			ip.usedBlobs++

			stats.size.used += size
			stats.blobs.used++
		default: // unused blob
			ip.unusedSize += size
			ip.unusedBlobs++

			stats.size.unused += size
			stats.blobs.unused++
		}
		if !blob.IsCompressed() {
			ip.uncompressed = true
		}
		// update indexPack
		indexPack[blob.PackID] = ip
	})

	// if duplicate blobs exist, those will be set to either "used" or "unused":
	// - mark only one occurence of duplicate blobs as used
	// - if there are already some used blobs in a pack, possibly mark duplicates in this pack as "used"
	// - if there are no used blobs in a pack, possibly mark duplicates as "unused"
	if hasDuplicates {
		// iterate again over all blobs in index (this is pretty cheap, all in-mem)
		idx.Each(ctx, func(blob restic.PackedBlob) {
			bh := blob.BlobHandle
			count, ok := usedBlobs[bh]
			// skip non-duplicate, aka. normal blobs
			// count == 0 is used to mark that this was a duplicate blob with only a single occurence remaining
			if !ok || count == 1 {
				return
			}

			ip := indexPack[blob.PackID]
			size := uint64(blob.Length)
			switch {
			case ip.usedBlobs > 0, count == 0:
				// other used blobs in pack or "last" occurence ->  transition to used
				ip.usedSize += size
				ip.usedBlobs++
				ip.unusedSize -= size
				ip.unusedBlobs--
				// same for the global statistics
				stats.size.used += size
				stats.blobs.used++
				stats.size.duplicate -= size
				stats.blobs.duplicate--
				// let other occurences remain marked as unused
				usedBlobs[bh] = 1
			default:
				// remain unused and decrease counter
				count--
				if count == 1 {
					// setting count to 1 would lead to forgetting that this blob had duplicates
					// thus use the special value zero. This will select the last instance of the blob for keeping.
					count = 0
				}
				usedBlobs[bh] = count
			}
			// update indexPack
			indexPack[blob.PackID] = ip
		})
	}

	// Sanity check. If no duplicates exist, all blobs have value 1. After handling
	// duplicates, this also applies to duplicates.
	for _, count := range usedBlobs {
		if count != 1 {
			panic("internal error during blob selection")
		}
	}

	return usedBlobs, indexPack, nil
}

func decidePackAction(ctx context.Context, opts PruneOptions, repo restic.Repository, indexPack map[restic.ID]packInfo, stats *pruneStats, spr *wsTaskInfo.Sprintf) (prunePlan, error) {
	removePacksFirst := restic.NewIDSet()
	removePacks := restic.NewIDSet()
	repackPacks := restic.NewIDSet()

	var repackCandidates []packInfoWithID
	var repackSmallCandidates []packInfoWithID
	repoVersion := repo.Config().Version
	// only repack very small files by default
	targetPackSize := repo.PackSize() / 25
	if opts.RepackSmall {
		// consider files with at least 80% of the target size as large enough
		targetPackSize = repo.PackSize() / 5 * 4
	}

	// loop over all packs and decide what to do
	bar := newProgressMax(true, uint64(len(indexPack)), "packs processed", spr)
	err := repo.List(ctx, restic.PackFile, func(id restic.ID, packSize int64) error {
		p, ok := indexPack[id]
		if !ok {
			// Pack was not referenced in index and is not used  => immediately remove!
			spr.AppendByForce(wsTaskInfo.Info, fmt.Sprintf("will remove pack %v as it is unused and not indexed\n", id.Str()), false)
			removePacksFirst.Insert(id)
			stats.size.unref += uint64(packSize)
			return nil
		}

		if p.unusedSize+p.usedSize != uint64(packSize) && p.usedBlobs != 0 {
			// Pack size does not fit and pack is needed => error
			// If the pack is not needed, this is no error, the pack can
			// and will be simply removed, see below.
			spr.AppendByForce(wsTaskInfo.Info, fmt.Sprintf("pack %s: calculated size %d does not match real size %d\nRun 'restic repair index'.\n",
				id.Str(), p.unusedSize+p.usedSize, packSize), false)
			return errorSizeNotMatching
		}

		// statistics
		switch {
		case p.usedBlobs == 0:
			stats.packs.unused++
		case p.unusedBlobs == 0:
			stats.packs.used++
		default:
			stats.packs.partlyUsed++
		}

		if p.uncompressed {
			stats.size.uncompressed += p.unusedSize + p.usedSize
		}
		mustCompress := false
		if repoVersion >= 2 {
			// repo v2: always repack tree blobs if uncompressed
			// compress data blobs if requested
			mustCompress = (p.tpe == restic.TreeBlob || opts.RepackUncompressed) && p.uncompressed
		}

		// decide what to do
		switch {
		case p.usedBlobs == 0:
			// All blobs in pack are no longer used => remove pack!
			removePacks.Insert(id)
			stats.blobs.remove += p.unusedBlobs
			stats.size.remove += p.unusedSize

		case opts.RepackCachableOnly && p.tpe == restic.DataBlob:
			// if this is a data pack and --repack-cacheable-only is set => keep pack!
			stats.packs.keep++

		case p.unusedBlobs == 0 && p.tpe != restic.InvalidBlob && !mustCompress:
			if packSize >= int64(targetPackSize) {
				// All blobs in pack are used and not mixed => keep pack!
				stats.packs.keep++
			} else {
				repackSmallCandidates = append(repackSmallCandidates, packInfoWithID{ID: id, packInfo: p, mustCompress: mustCompress})
			}

		default:
			// all other packs are candidates for repacking
			repackCandidates = append(repackCandidates, packInfoWithID{ID: id, packInfo: p, mustCompress: mustCompress})
		}

		delete(indexPack, id)
		bar.Add(1)
		return nil
	})
	bar.Done()
	if err != nil {
		return prunePlan{}, err
	}

	// At this point indexPacks contains only missing packs!

	// missing packs that are not needed can be ignored
	ignorePacks := restic.NewIDSet()
	for id, p := range indexPack {
		if p.usedBlobs == 0 {
			ignorePacks.Insert(id)
			stats.blobs.remove += p.unusedBlobs
			stats.size.remove += p.unusedSize
			delete(indexPack, id)
		}
	}

	if len(indexPack) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("The index references %d needed pack files which are missing from the repository:\n", len(indexPack)))
		for id := range indexPack {
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("  %v\n", id))
		}
		return prunePlan{}, errorPacksMissing
	}
	if len(ignorePacks) != 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("Missing but unneeded pack files are referenced in the index, will be repaired\n"))
		for id := range ignorePacks {
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("will forget missing pack file %v\n", id))
		}
	}

	if len(repackSmallCandidates) < 10 {
		// too few small files to be worth the trouble, this also prevents endlessly repacking
		// if there is just a single pack file below the target size
		stats.packs.keep += uint(len(repackSmallCandidates))
	} else {
		repackCandidates = append(repackCandidates, repackSmallCandidates...)
	}

	// Sort repackCandidates such that packs with highest ratio unused/used space are picked first.
	// This is equivalent to sorting by unused / total space.
	// Instead of unused[i] / used[i] > unused[j] / used[j] we use
	// unused[i] * used[j] > unused[j] * used[i] as uint32*uint32 < uint64
	// Moreover packs containing trees and too small packs are sorted to the beginning
	sort.Slice(repackCandidates, func(i, j int) bool {
		pi := repackCandidates[i].packInfo
		pj := repackCandidates[j].packInfo
		switch {
		case pi.tpe != restic.DataBlob && pj.tpe == restic.DataBlob:
			return true
		case pj.tpe != restic.DataBlob && pi.tpe == restic.DataBlob:
			return false
		case pi.unusedSize+pi.usedSize < uint64(targetPackSize) && pj.unusedSize+pj.usedSize >= uint64(targetPackSize):
			return true
		case pj.unusedSize+pj.usedSize < uint64(targetPackSize) && pi.unusedSize+pi.usedSize >= uint64(targetPackSize):
			return false
		}
		return pi.unusedSize*pj.usedSize > pj.unusedSize*pi.usedSize
	})

	repack := func(id restic.ID, p packInfo) {
		repackPacks.Insert(id)
		stats.blobs.repack += p.unusedBlobs + p.usedBlobs
		stats.size.repack += p.unusedSize + p.usedSize
		stats.blobs.repackrm += p.unusedBlobs
		stats.size.repackrm += p.unusedSize
		if p.uncompressed {
			stats.size.uncompressed -= p.unusedSize + p.usedSize
		}
	}

	// calculate limit for number of unused bytes in the repo after repacking
	maxUnusedSizeAfter := opts.maxUnusedBytes(stats.size.used)

	for _, p := range repackCandidates {
		reachedUnusedSizeAfter := (stats.size.unused-stats.size.remove-stats.size.repackrm < maxUnusedSizeAfter)
		reachedRepackSize := stats.size.repack+p.unusedSize+p.usedSize >= opts.MaxRepackBytes
		packIsLargeEnough := p.unusedSize+p.usedSize >= uint64(targetPackSize)

		switch {
		case reachedRepackSize:
			stats.packs.keep++

		case p.tpe != restic.DataBlob, p.mustCompress:
			// repacking non-data packs / uncompressed-trees is only limited by repackSize
			repack(p.ID, p.packInfo)

		case reachedUnusedSizeAfter && packIsLargeEnough:
			// for all other packs stop repacking if tolerated unused size is reached.
			stats.packs.keep++

		default:
			repack(p.ID, p.packInfo)
		}
	}

	stats.packs.unref = uint(len(removePacksFirst))
	stats.packs.repack = uint(len(repackPacks))
	stats.packs.remove = uint(len(removePacks))

	if repo.Config().Version < 2 {
		// compression not supported for repository format version 1
		stats.size.uncompressed = 0
	}

	return prunePlan{removePacksFirst: removePacksFirst,
		removePacks: removePacks,
		repackPacks: repackPacks,
		ignorePacks: ignorePacks,
	}, nil
}

func getUsedBlobs(ctx context.Context, repo restic.Repository, ignoreSnapshots restic.IDSet, spr *wsTaskInfo.Sprintf) (usedBlobs restic.CountedBlobSet, err error) {

	var snapshotTrees restic.IDs
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("loading all snapshots...\n"))
	err = restic.ForAllSnapshots(ctx, repo.Backend(), repo, ignoreSnapshots,
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

	usedBlobs = restic.NewCountedBlobSet()

	pro := newProgressMax(true, uint64(len(snapshotTrees)), "snapshots", spr)
	defer pro.Done()

	err = restic.FindUsedBlobs(ctx, repo, snapshotTrees, usedBlobs, pro)
	if err != nil {
		if repo.Backend().IsNotExist(err) {
			return nil, errors.Fatal("unable to load a tree from the repo: " + err.Error())
		}

		return nil, err
	}
	return usedBlobs, nil
}
