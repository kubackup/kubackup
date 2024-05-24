//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package resticProxy

import (
	systemFuse "bazil.org/fuse"
	"bazil.org/fuse/fs"
	"fmt"
	"github.com/kubackup/kubackup/internal/server"
	fileutil "github.com/kubackup/kubackup/pkg/file"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fuse"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"os"
	"strconv"
	"strings"
)

// MountOptions collects all options for the mount command.
type MountOptions struct {
	OwnerRoot            bool
	AllowOther           bool
	NoDefaultPermissions bool
	Hosts                []string
	Tags                 restic.TagLists
	Paths                []string
	SnapshotTemplate     string
}

func runMount(opts MountOptions, repoid int) error {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return err
	}
	repo := repoHandler.repo
	if opts.SnapshotTemplate == "" {
		return fmt.Errorf("snapshot template string cannot be empty")
	}
	if strings.ContainsAny(opts.SnapshotTemplate, `\/`) {
		return fmt.Errorf("snapshot template string contains a slash (/) or backslash (\\) character")
	}
	server.Logger().Info("start mount")
	defer server.Logger().Info("finish mount")

	mountpoint := "/tmp/backup_" + strconv.Itoa(repoid)

	if exist := fileutil.Exist(mountpoint); exist {
		fileutil.Mkdir(mountpoint, os.ModeDir)
	}
	mountOptions := []systemFuse.MountOption{
		systemFuse.ReadOnly(),
		systemFuse.FSName("backup_" + strconv.Itoa(repoid)),
		systemFuse.MaxReadahead(128 * 1024),
	}

	if opts.AllowOther {
		mountOptions = append(mountOptions, systemFuse.AllowOther())

		// let the kernel check permissions unless it is explicitly disabled
		if !opts.NoDefaultPermissions {
			mountOptions = append(mountOptions, systemFuse.DefaultPermissions())
		}
	}
	c, err := systemFuse.Mount(mountpoint, mountOptions...)
	if err != nil {
		return err
	}

	systemFuse.Debug = func(msg interface{}) {
		server.Logger().Debugf("fuse: %v", msg)
	}

	cfg := fuse.Config{
		OwnerIsRoot:      opts.OwnerRoot,
		Hosts:            opts.Hosts,
		Tags:             opts.Tags,
		Paths:            opts.Paths,
		SnapshotTemplate: opts.SnapshotTemplate,
	}
	root := fuse.NewRoot(repo, cfg)

	server.Logger().Infof("serving mount at %v", mountpoint)
	err = fs.Serve(c, root)
	if err != nil {
		return err
	}
	<-c.Ready
	return c.MountError
}

func umount(mountpoint string) error {
	return systemFuse.Unmount(mountpoint)
}
