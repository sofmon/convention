package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
	convDB "github.com/sofmon/convention/lib/db"
)

type JobID string

type JobState string

type JobFunc func(convCtx.Context) error

type job struct {
	ID          JobID         `json:"id"`
	NextRunAt   time.Time     `json:"next_run_at"`
	RepeatEvery time.Duration `json:"repeat_every"`

	f JobFunc `json:"-"`
}

func (x job) DBKey() convDB.Key[JobID, JobID] {
	return convDB.Key[JobID, JobID]{
		ID:       x.ID,
		ShardKey: x.ID,
	}
}

var (
	jobs   map[convAuth.Tenant]map[JobID]job
	mut    sync.Mutex
	cancel context.CancelFunc
	jobsDB convDB.ObjectSetReady[job, JobID, JobID]
	wakeUp chan struct{}
)

func Register(ctx convCtx.Context, tenant convAuth.Tenant, jid JobID, startAt time.Time, repeatEvery time.Duration, fn JobFunc) (err error) {

	if jobsDB == nil {
		err = fmt.Errorf("job runner is not initialised - call Initialise first")
		return
	}

	mut.Lock()
	defer mut.Unlock()

	if jobs == nil {
		jobs = make(map[convAuth.Tenant]map[JobID]job)
	}

	if _, ok := jobs[tenant]; !ok {
		jobs[tenant] = make(map[JobID]job)
	}

	if _, ok := jobs[tenant][jid]; ok {
		err = fmt.Errorf("job with id %s already exists", jid)
		return
	}

	savedJob, err := jobsDB.Tenant(tenant).SelectByID(ctx, jid)
	if err != nil {
		return
	}

	if savedJob == nil {
		savedJob = &job{
			ID:          jid,
			NextRunAt:   startAt,
			RepeatEvery: repeatEvery,
			f:           fn,
		}

		err = jobsDB.Tenant(tenant).Insert(ctx, *savedJob)
		if err != nil {
			return
		}
	} else {
		savedJob.f = fn
		savedJob.NextRunAt = startAt
		savedJob.RepeatEvery = repeatEvery
	}

	jobs[tenant][jid] = *savedJob

	// Wake up the background loop to evaluate the new job
	select {
	case wakeUp <- struct{}{}:
	default:
	}

	return
}

func Unregister(ctx convCtx.Context, tenant convAuth.Tenant, jid JobID) (err error) {

	if jobsDB == nil {
		err = fmt.Errorf("job runner is not initialised - call Initialise first")
		return
	}

	mut.Lock()
	defer mut.Unlock()

	if _, ok := jobs[tenant]; !ok {
		err = fmt.Errorf("tenant %s does not exist", tenant)
		return
	}

	if _, ok := jobs[tenant][jid]; !ok {
		err = fmt.Errorf("job with id %s does not exist", jid)
		return
	}

	delete(jobs[tenant], jid)

	return jobsDB.Tenant(tenant).Delete(ctx, jid)
}

func Cancel() (err error) {
	mut.Lock()
	defer mut.Unlock()

	if cancel == nil {
		err = fmt.Errorf("job runner is not running")
		return
	}

	cancel()
	cancel = nil

	return
}

func Initialise(ctx convCtx.Context, vault convDB.Vault) (err error) {
	mut.Lock()
	defer mut.Unlock()

	if cancel != nil {
		err = fmt.Errorf("job runner is already running")
		return
	}

	jobsDB = convDB.NewObjectSet[job, JobID, JobID](vault).Ready()

	wakeUp = make(chan struct{}, 1)

	ctx.Context, cancel = context.WithCancel(ctx.Context)

	go background(ctx)

	return
}

func background(ctx convCtx.Context) {

	const syncInterval = 1 * time.Minute

	lastSync := time.Time{} // zero value forces immediate first sync

	for {
		now := time.Now().UTC()

		// (1) Sync with database periodically
		if now.Sub(lastSync) >= syncInterval {
			syncJobsFromDB(ctx)
			lastSync = time.Now().UTC()
		}

		// (2) & (3) Execute due jobs and determine next wake-up
		nextWakeUp := executeAndSchedule(ctx)

		// Compute sleep duration
		timeUntilNextSync := syncInterval - time.Since(lastSync)
		sleepDuration := timeUntilNextSync

		if !nextWakeUp.IsZero() {
			timeUntilNextJob := time.Until(nextWakeUp)
			if timeUntilNextJob < sleepDuration {
				sleepDuration = timeUntilNextJob
			}
		}

		if sleepDuration < 1*time.Second {
			sleepDuration = 1 * time.Second
		}

		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-wakeUp:
			timer.Stop()
		case <-timer.C:
		}
	}
}

func syncJobsFromDB(ctx convCtx.Context) {

	mut.Lock()
	defer mut.Unlock()

	if jobs == nil {
		return
	}

	for tenant, tenantJobs := range jobs {

		dbJobs, err := jobsDB.Tenant(tenant).SelectAll(ctx)
		if err != nil {
			ctx.Logger().Error("failed to sync jobs from database",
				"tenant", string(tenant),
				"error", err.Error(),
			)
			continue
		}

		dbJobMap := make(map[JobID]job, len(dbJobs))
		for _, dj := range dbJobs {
			dbJobMap[dj.ID] = dj
		}

		// Merge DB state into memory, preserving f
		for jid, memJob := range tenantJobs {
			if dbJob, ok := dbJobMap[jid]; ok {
				dbJob.f = memJob.f
				tenantJobs[jid] = dbJob
			}
		}

		// Add DB-only jobs (f will be nil — tracked but not executable by this instance)
		for jid, dbJob := range dbJobMap {
			if _, ok := tenantJobs[jid]; !ok {
				tenantJobs[jid] = dbJob
			}
		}
	}
}

func executeAndSchedule(ctx convCtx.Context) (nextWakeUp time.Time) {

	type dueJob struct {
		tenant convAuth.Tenant
		job    job
	}

	mut.Lock()
	var dueJobs []dueJob
	now := time.Now().UTC()

	for tenant, tenantJobs := range jobs {
		for _, j := range tenantJobs {
			if j.NextRunAt.IsZero() {
				continue
			}
			if !j.NextRunAt.After(now) && j.f != nil {
				dueJobs = append(dueJobs, dueJob{tenant: tenant, job: j})
			}
		}
	}
	mut.Unlock()

	// Execute due jobs outside the lock
	for _, dj := range dueJobs {
		executeJob(ctx, dj.tenant, dj.job)
	}

	// Re-scan for accurate next wake-up
	mut.Lock()
	defer mut.Unlock()

	now = time.Now().UTC()
	for _, tenantJobs := range jobs {
		for _, j := range tenantJobs {
			if j.NextRunAt.IsZero() {
				continue
			}
			if j.NextRunAt.After(now) {
				if nextWakeUp.IsZero() || j.NextRunAt.Before(nextWakeUp) {
					nextWakeUp = j.NextRunAt
				}
			}
		}
	}

	return
}

func executeJob(ctx convCtx.Context, tenant convAuth.Tenant, j job) {

	// Guard against zero/negative repeat interval
	if j.RepeatEvery <= 0 {
		ctx.Logger().Error("job has invalid repeat interval, skipping",
			"tenant", string(tenant),
			"job_id", string(j.ID),
			"repeat_every", j.RepeatEvery.String(),
		)
		return
	}

	// Attempt to acquire DB lock (non-blocking)
	lock, err := jobsDB.Tenant(tenant).Lock(ctx, j, fmt.Sprintf("executing job %s", j.ID))
	if err != nil {
		ctx.Logger().Error("failed to acquire lock for job",
			"tenant", string(tenant),
			"job_id", string(j.ID),
			"error", err.Error(),
		)
		return
	}
	if lock == nil {
		// Another instance holds the lock
		return
	}

	defer func() {
		if unlockErr := lock.Unlock(); unlockErr != nil {
			ctx.Logger().Error("failed to unlock job",
				"tenant", string(tenant),
				"job_id", string(j.ID),
				"error", unlockErr.Error(),
			)
		}
	}()

	// Execute job function with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				ctx.Logger().Error("job panicked",
					"tenant", string(tenant),
					"job_id", string(j.ID),
					"panic", fmt.Sprintf("%v", r),
				)
			}
		}()

		if err := j.f(ctx); err != nil {
			ctx.Logger().Error("job execution failed",
				"tenant", string(tenant),
				"job_id", string(j.ID),
				"error", err.Error(),
			)
		}
	}()

	// Compute next run time, advancing past any missed intervals
	now := time.Now().UTC()
	nextRunAt := j.NextRunAt.Add(j.RepeatEvery)
	for !nextRunAt.After(now) {
		nextRunAt = nextRunAt.Add(j.RepeatEvery)
	}

	// Update in-memory state
	mut.Lock()
	if tenantJobs, ok := jobs[tenant]; ok {
		if memJob, ok := tenantJobs[j.ID]; ok {
			memJob.NextRunAt = nextRunAt
			tenantJobs[j.ID] = memJob
		}
	}
	mut.Unlock()

	// Update database state
	updatedJob := j
	updatedJob.NextRunAt = nextRunAt
	if err := jobsDB.Tenant(tenant).Update(ctx, updatedJob); err != nil {
		ctx.Logger().Error("failed to update job next run time in database",
			"tenant", string(tenant),
			"job_id", string(j.ID),
			"error", err.Error(),
		)
	}
}
