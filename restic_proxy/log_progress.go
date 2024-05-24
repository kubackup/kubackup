package resticProxy

import (
	"fmt"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/progress"
	"github.com/kubackup/kubackup/pkg/utils"
	"time"
)

// newProgressMax returns a progress.Counter that prints to stdout.
func newProgressMax(show bool, max uint64, description string, spr *wsTaskInfo.Sprintf) *progress.Counter {
	if !show {
		return nil
	}
	return progress.New(spr.MinUpdatePause, max, func(v uint64, max uint64, d time.Duration, final bool) {
		var status string
		if max == 0 {
			status = fmt.Sprintf("[%s]          %d %s", utils.FormatDuration(d), v, description)
		} else {
			status = fmt.Sprintf("[%s] %s  %d / %d %s",
				utils.FormatDuration(d), utils.FormatPercent(v, max), v, max, description)
		}
		spr.AppendForClear(wsTaskInfo.Info, status, final)
	})
}
