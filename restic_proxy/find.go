package resticProxy

import (
	"context"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
)

// FindFilteredSnapshots yields Snapshots, either given explicitly by `snapshotIDs` or filtered from the list of all snapshots.
func FindFilteredSnapshots(ctx context.Context, be restic.Lister, loader restic.LoaderUnpacked, f *restic.SnapshotFilter, snapshotIDs []string) <-chan *restic.Snapshot {
	out := make(chan *restic.Snapshot)
	go func() {
		defer close(out)
		be, err := backend.MemorizeList(ctx, be, restic.SnapshotFile)
		if err != nil {
			server.Logger().Warnf("could not load snapshots: %v\n", err)
			return
		}

		err = f.FindAll(ctx, be, loader, snapshotIDs, func(id string, sn *restic.Snapshot, err error) error {
			if err != nil {
				server.Logger().Warnf("Ignoring %q: %v\n", id, err)
			} else {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- sn:
				}
			}
			return nil
		})
		if err != nil {
			server.Logger().Warnf("could not load snapshots: %v\n", err)
		}
	}()
	return out
}
