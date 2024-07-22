package resticProxy

import (
	"context"
	"encoding/json"
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
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/table"
	"gopkg.in/tomb.v2"
	"io"
	"os"
	"sort"
	"strconv"
)

type ForgetPolicyCount int

var ErrNegativePolicyCount = errors.New("negative values not allowed, use 'unlimited' instead")

func (c *ForgetPolicyCount) Set(s string) error {
	switch s {
	case "unlimited":
		*c = -1
	default:
		val, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}
		if val < 0 {
			return ErrNegativePolicyCount
		}
		*c = ForgetPolicyCount(val)
	}

	return nil
}

func (c *ForgetPolicyCount) String() string {
	switch *c {
	case -1:
		return "unlimited"
	default:
		return strconv.FormatInt(int64(*c), 10)
	}
}

func (c *ForgetPolicyCount) Type() string {
	return "n"
}

// ForgetOptions collects all options for the forget command.
type ForgetOptions struct {
	Last          ForgetPolicyCount
	Hourly        ForgetPolicyCount
	Daily         ForgetPolicyCount
	Weekly        ForgetPolicyCount
	Monthly       ForgetPolicyCount
	Yearly        ForgetPolicyCount
	Within        restic.Duration
	WithinHourly  restic.Duration
	WithinDaily   restic.Duration
	WithinWeekly  restic.Duration
	WithinMonthly restic.Duration
	WithinYearly  restic.Duration
	KeepTags      restic.TagLists

	restic.SnapshotFilter
	Compact bool //use compact output format

	// Grouping
	GroupBy restic.SnapshotGroupByOptions
	DryRun  bool
	Prune   bool // automatically run the 'prune' command if snapshots have been removed
}

func RunForget(opts ForgetOptions, repoid int, snapshotids []string) (int, error) {
	hostname, err := os.Hostname()
	opts.Hosts = []string{hostname}

	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return 0, err
	}
	repo := repoHandler.repo

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
		Type:         operationModel.FORGET_TYPE,
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
		err := forget(opts, ctx, repo, snapshotids, spr)
		status = repoModel.StatusNone
		if err != nil {
			spr.Append(wsTaskInfo.Error, err.Error())
			status = repoModel.StatusErr
		} else {
			status = repoModel.StatusRun
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

func RunForgetSync(opts ForgetOptions, repoid int, snapshotids []string) error {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return err
	}
	repo := repoHandler.repo

	ctx := context.Background()
	lock, err := lockRepoExclusive(ctx, repo)
	defer unlockRepo(lock)
	if err != nil {
		return err
	}
	logTask := log.LogInfo{}
	logTask.SetId(0)
	spr := wsTaskInfo.NewSprintf(&logTask)
	err = forget(opts, ctx, repo, snapshotids, spr)
	if err != nil {
		return err
	}
	return nil
}
func forget(opts ForgetOptions, ctx context.Context, repo *repository.Repository, snapshotids []string, spr *wsTaskInfo.Sprintf) error {

	var snapshots restic.Snapshots
	removeSnIDs := restic.NewIDSet()

	for sn := range FindFilteredSnapshots(ctx, repo.Backend(), repo, &opts.SnapshotFilter, snapshotids) {
		snapshots = append(snapshots, sn)
	}
	if len(snapshots) <= 0 {
		spr.Append(wsTaskInfo.Error, "快照不存在！")
		return fmt.Errorf("快照不存在！")
	}
	if len(snapshotids) > 0 {
		// When explicit snapshots args are given, remove them immediately.
		for _, sn := range snapshots {
			removeSnIDs.Insert(*sn.ID())
		}
	} else {
		snapshotGroups, _, err := restic.GroupSnapshots(snapshots, opts.GroupBy)
		if err != nil {
			return err
		}

		policy := restic.ExpirePolicy{
			Last:          int(opts.Last),
			Hourly:        int(opts.Hourly),
			Daily:         int(opts.Daily),
			Weekly:        int(opts.Weekly),
			Monthly:       int(opts.Monthly),
			Yearly:        int(opts.Yearly),
			Within:        opts.Within,
			WithinHourly:  opts.WithinHourly,
			WithinDaily:   opts.WithinDaily,
			WithinWeekly:  opts.WithinWeekly,
			WithinMonthly: opts.WithinMonthly,
			WithinYearly:  opts.WithinYearly,
			Tags:          opts.KeepTags,
		}

		if policy.Empty() && len(snapshotids) == 0 {
			spr.Append(wsTaskInfo.Warning, fmt.Sprintf("no policy was specified, no snapshots will be removed\n"))
		}

		if !policy.Empty() {
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("Applying Policy: %v\n", policy))
			for k, snapshotGroup := range snapshotGroups {
				var key restic.SnapshotGroupKey
				if json.Unmarshal([]byte(k), &key) != nil {
					return err
				}
				keep, remove, reasons := restic.ApplyPolicy(snapshotGroup, policy)

				if len(keep) != 0 {
					spr.Append(wsTaskInfo.Info, fmt.Sprintf("keep %d snapshots:\n", len(keep)))
					PrintSnapshots(spr, keep, reasons, opts.Compact)
					spr.Append(wsTaskInfo.Info, fmt.Sprintf("\n"))
				}
				if len(remove) != 0 {
					spr.Append(wsTaskInfo.Info, fmt.Sprintf("remove %d snapshots:\n", len(remove)))
					PrintSnapshots(spr, remove, nil, opts.Compact)
					spr.Append(wsTaskInfo.Info, fmt.Sprintf("\n"))
				}
				for _, sn := range remove {
					removeSnIDs.Insert(*sn.ID())
				}
			}
		}
	}

	if len(removeSnIDs) > 0 {
		err := DeleteFilesChecked(spr, ctx, repo, removeSnIDs, restic.SnapshotFile)
		if err != nil {
			return err
		}
	}

	if len(removeSnIDs) > 0 && opts.Prune {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("%d snapshots have been removed, running prune\n", len(removeSnIDs)))
		pruneOptions := PruneOptions{
			MaxUnused: "5%",
		}
		err := verifyPruneOptions(&pruneOptions)
		if err != nil {
			return err
		}
		err = runPruneWithRepo(pruneOptions, ctx, repo, removeSnIDs, spr)
		if err != nil {
			return err
		}
		return repo.LoadIndex(ctx, nil)
	}
	return nil
}

// PrintSnapshots prints a text table of the snapshots in list to stdout.
func PrintSnapshots(spr *wsTaskInfo.Sprintf, list restic.Snapshots, reasons []restic.KeepReason, compact bool) {
	// keep the reasons a snasphot is being kept in a map, so that it doesn't
	// get lost when the list of snapshots is sorted
	keepReasons := make(map[restic.ID]restic.KeepReason, len(reasons))
	if len(reasons) > 0 {
		for i, sn := range list {
			id := sn.ID()
			keepReasons[*id] = reasons[i]
		}
	}

	// always sort the snapshots so that the newer ones are listed last
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Time.Before(list[j].Time)
	})

	// Determine the max widths for host and tag.
	maxHost, maxTag := 10, 6
	for _, sn := range list {
		if len(sn.Hostname) > maxHost {
			maxHost = len(sn.Hostname)
		}
		for _, tag := range sn.Tags {
			if len(tag) > maxTag {
				maxTag = len(tag)
			}
		}
	}

	tab := table.New()

	if compact {
		tab.AddColumn("ID", "{{ .ID }}")
		tab.AddColumn("Time", "{{ .Timestamp }}")
		tab.AddColumn("Host", "{{ .Hostname }}")
		tab.AddColumn("Tags  ", `{{ join .Tags "\n" }}`)
	} else {
		tab.AddColumn("ID", "{{ .ID }}")
		tab.AddColumn("Time", "{{ .Timestamp }}")
		tab.AddColumn("Host      ", "{{ .Hostname }}")
		tab.AddColumn("Tags      ", `{{ join .Tags "," }}`)
		if len(reasons) > 0 {
			tab.AddColumn("Reasons", `{{ join .Reasons "\n" }}`)
		}
		tab.AddColumn("Paths", `{{ join .Paths "\n" }}`)
	}

	type snapshot struct {
		ID        string
		Timestamp string
		Hostname  string
		Tags      []string
		Reasons   []string
		Paths     []string
	}

	var multiline bool
	for _, sn := range list {
		data := snapshot{
			ID:        sn.ID().Str(),
			Timestamp: sn.Time.Local().Format(TimeFormat),
			Hostname:  sn.Hostname,
			Tags:      sn.Tags,
			Paths:     sn.Paths,
		}

		if len(reasons) > 0 {
			id := sn.ID()
			data.Reasons = keepReasons[*id].Matches
		}

		if len(sn.Paths) > 1 && !compact {
			multiline = true
		}

		tab.AddRow(data)
	}

	tab.AddFooter(fmt.Sprintf("%d snapshots", len(list)))

	if multiline {
		// print an additional blank line between snapshots

		var last int
		tab.PrintData = func(w io.Writer, idx int, s string) error {
			var err error
			if idx == last {
				_, err = fmt.Fprintf(w, "%s\n", s)
			} else {
				_, err = fmt.Fprintf(w, "\n%s\n", s)
			}
			last = idx
			return err
		}
	}
	err := tab.Write(spr)
	if err != nil {
		spr.Append(wsTaskInfo.Error, fmt.Sprintf("error printing: %v\n", err))
	}
}
