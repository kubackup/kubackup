package resticProxy

import (
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
)

func RunInit(gopts GlobalOptions) error {
	repo, err := ReadRepo(gopts)
	if err != nil {
		return err
	}
	be, err := create(repo, gopts, gopts.extended)
	if err != nil {
		return errors.Fatalf("create repository at %s failed: %v\n", StripPassword(gopts.Repo), err)
	}
	s := repository.New(be)
	err = s.Init(gopts.ctx, gopts.password, nil)
	if err != nil {
		return errors.Fatalf("create key in repository at %s failed: %v\n", StripPassword(gopts.Repo), err)
	}
	return nil
}
