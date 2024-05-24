package resticProxy

import (
	"encoding/json"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
)

func RunCat(repoid int, tpe, idstr string) (string, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return "", err
	}
	gopts := repoHandler.gopts
	repo := repoHandler.repo
	if idstr == "" && tpe != "masterkey" && tpe != "config" {
		return "", errors.Fatal("type or ID not specified")
	}
	var id restic.ID
	if tpe != "masterkey" && tpe != "config" {
		id, err = restic.ParseID(idstr)
		if err != nil {
			if tpe != "snapshot" {
				return "", errors.Fatalf("unable to parse ID: %v\n", err)
			}

			// find snapshot id with prefix
			id, err = restic.FindSnapshot(gopts.ctx, repo, idstr)
			if err != nil {
				return "", errors.Fatalf("could not find snapshot: %v\n", err)
			}
		}
	}

	switch tpe {
	case "config":
		buf, err := json.MarshalIndent(repo.Config(), "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	case "index":
		buf, err := repo.LoadAndDecrypt(gopts.ctx, nil, restic.IndexFile, id)
		if err != nil {
			return "", err
		}

		return string(buf), nil
	case "snapshot":
		sn := &restic.Snapshot{}
		err = repo.LoadJSONUnpacked(gopts.ctx, restic.SnapshotFile, id, sn)
		if err != nil {
			return "", err
		}

		buf, err := json.MarshalIndent(&sn, "", "  ")
		if err != nil {
			return "", err
		}

		return string(buf), nil
	case "key":
		h := restic.Handle{Type: restic.KeyFile, Name: id.String()}
		buf, err := backend.LoadAll(gopts.ctx, nil, repo.Backend(), h)
		if err != nil {
			return "", err
		}

		key := &repository.Key{}
		err = json.Unmarshal(buf, key)
		if err != nil {
			return "", err
		}

		buf, err = json.MarshalIndent(&key, "", "  ")
		if err != nil {
			return "", err
		}

		return string(buf), nil
	case "masterkey":
		buf, err := json.MarshalIndent(repo.Key(), "", "  ")
		if err != nil {
			return "", err
		}

		return string(buf), nil
	case "lock":
		lock, err := restic.LoadLock(gopts.ctx, repo, id)
		if err != nil {
			return "", err
		}

		buf, err := json.MarshalIndent(&lock, "", "  ")
		if err != nil {
			return "", err
		}

		return string(buf), nil
	default:
		return "", errors.Fatal("invalid type")
	}
}
