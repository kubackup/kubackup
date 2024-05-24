package cron

type PruneJob struct {
	PlanId int
}

func (b PruneJob) Run() {

}

func (b PruneJob) GetType() int {
	return JOB_TYPE_PRUNE
}

var _ BaseJob = &PruneJob{}
