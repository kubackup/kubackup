package cron

type SystemJob func()

func (f SystemJob) Run() { f() }

func (f SystemJob) GetType() int {
	return JOB_TYPE_SYSTEM
}
