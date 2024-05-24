package restore

import (
	"context"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/archiver"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/signals"
	"time"
)

type RestoreProgressReporter interface {
	CompleteItem(item string, s archiver.ScanStats)
	ScannerError(err error) error
	Error(item string, err error) error
	ReportTotal(item string, total Stats)
	SetMinUpdatePause(d time.Duration)
	Run(ctx context.Context) error
	Finish(snapshotID string)
}

type ProgressPrinter interface {
	Update(total, s Counter, avgSpeed uint64, errors uint, start time.Time, secs uint64)
	Error(item string, err error) error
	ScannerError(err error) error
	ReportTotal(start time.Time, total Counter)
	Finish(snapshotID string, start time.Time, summary *Counter)
}

type Stats struct {
	TotalSize      uint64
	TotalFileCount uint
}

type Counter struct {
	Files uint
	Bytes uint64
}

type RestoreProgress struct {
	MinUpdatePause time.Duration
	start          time.Time
	errCh          chan struct{}
	totalCh        chan Counter
	processedCh    chan Counter
	closed         chan struct{}
	summary        *Counter
	printer        ProgressPrinter
}

func NewProgress(printer ProgressPrinter) *RestoreProgress {
	return &RestoreProgress{
		// limit to 60fps by default
		MinUpdatePause: time.Second / 60,
		start:          time.Now(),
		errCh:          make(chan struct{}),
		totalCh:        make(chan Counter),
		processedCh:    make(chan Counter),
		closed:         make(chan struct{}),

		summary: &Counter{},
		printer: printer,
	}
}

func (p *RestoreProgress) Run(ctx context.Context) error {
	var (
		lastUpdate       time.Time
		total            Counter
		processed        Counter
		started          bool
		errors           uint
		secondsRemaining uint64
		avgSpeed         uint64
	)
	t := time.NewTicker(time.Second)
	signalsCh := signals.GetProgressChannel()
	defer t.Stop()
	defer close(p.closed)
	for {
		forceUpdate := false
		select {
		case <-ctx.Done():
			return nil
		case t, ok := <-p.totalCh:
			if ok {
				total = t
				started = true
			} else {
				// scan has finished
				p.totalCh = nil
				p.summary = &total
			}
		case s := <-p.processedCh:
			processed.Files = s.Files
			processed.Bytes = s.Bytes
			started = true
		case <-p.errCh:
			errors++
			started = true
		case <-t.C:
			if !started {
				continue
			}
			secs := float64(time.Since(p.start) / time.Second)
			if p.totalCh == nil {
				todo := float64(total.Bytes - processed.Bytes)
				if processed.Bytes > 0 {
					secondsRemaining = uint64(secs / float64(processed.Bytes) * todo)
				}
			}
			if processed.Bytes > 0 && secs > 0 {
				avgSpeed = uint64(float64(processed.Bytes) / secs)
			}
		case <-signalsCh:
			forceUpdate = true
		}

		// limit update frequency
		if !forceUpdate && (time.Since(lastUpdate) < p.MinUpdatePause || p.MinUpdatePause == 0) {
			continue
		}
		lastUpdate = time.Now()

		p.printer.Update(total, processed, avgSpeed, errors, p.start, secondsRemaining)
	}
}

func (p *RestoreProgress) ScannerError(err error) error {
	return p.printer.ScannerError(err)
}

func (p *RestoreProgress) Error(item string, err error) error {
	cbErr := p.printer.Error(item, err)
	return cbErr
}

func (p *RestoreProgress) CompleteItem(item string, s archiver.ScanStats) {
	if item == "" {
		select {
		case p.processedCh <- Counter{Files: s.Files, Bytes: s.Bytes}:
		case <-p.closed:
		}
	}
}

func (p *RestoreProgress) ReportTotal(item string, s Stats) {
	c := Counter{Files: s.TotalFileCount, Bytes: s.TotalSize}
	select {
	case p.totalCh <- c:
	case <-p.closed:
	}
	if item == "" {
		p.printer.ReportTotal(p.start, c)
		close(p.totalCh)
		return
	}
}

func (p *RestoreProgress) Finish(snapshotID string) {
	// wait for the status update goroutine to shut down
	<-p.closed
	p.printer.Finish(snapshotID, p.start, p.summary)
}

func (p *RestoreProgress) SetMinUpdatePause(d time.Duration) {
	p.MinUpdatePause = d
}
