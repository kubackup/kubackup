package resticProxy

import (
	"context"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/location"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"strconv"
)

func RunInit(ctx context.Context, gopts GlobalOptions) error {
	repo, err := ReadRepo(gopts)
	if err != nil {
		return err
	}
	be, err := create(ctx, repo, gopts, gopts.extended)
	if err != nil {
		return errors.Fatalf("create repository at %s failed: %v\n", location.StripPassword(gopts.backends, gopts.Repo), err)
	}
	s, err := repository.New(be, repository.Options{
		Compression: gopts.Compression,
		PackSize:    gopts.PackSize * 1024 * 1024,
	})
	if err != nil {
		return errors.Fatal(err.Error())
	}

	var version uint
	if gopts.RepositoryVersion == "latest" || gopts.RepositoryVersion == "" {
		version = restic.MaxRepoVersion
	} else if gopts.RepositoryVersion == "stable" {
		version = restic.StableRepoVersion
	} else {
		v, err := strconv.ParseUint(gopts.RepositoryVersion, 10, 32)
		if err != nil {
			return errors.Fatal("invalid repository version")
		}
		version = uint(v)
	}
	if version < restic.MinRepoVersion || version > restic.MaxRepoVersion {
		return errors.Fatalf("only repository versions between %v and %v are allowed", restic.MinRepoVersion, restic.MaxRepoVersion)
	}

	err = s.Init(ctx, version, gopts.password, nil)
	if err != nil {
		return errors.Fatalf("create key in repository at %s failed: %v\n", location.StripPassword(gopts.backends, gopts.Repo), err)
	}
	return nil
}
