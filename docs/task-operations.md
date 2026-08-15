# Task execution operations and queue migration gate

## Decision (2026-08-15)

The project currently keeps task execution in the application process. `TaskManager`
starts one goroutine per task, processes that task's pending items sequentially,
persists item and task state in PostgreSQL, and reclaims database tasks left in
`running` state after a restart.

The queue/worker migration is deferred for now. The current local deployment has
one backend instance, and the live PostgreSQL data inspected on 2026-08-15 had no
`tasks` or `task_items` rows. There is no observed backlog or processing pressure
that justifies introducing a broker or a second execution topology at this stage.

This is a capacity decision, not a claim that the current executor is safe for
horizontal scaling. `TaskManager.running` is process-local, and `StartTask` does
not atomically claim a task in PostgreSQL. Multiple backend replicas could
therefore execute the same task after concurrent starts or recovery.

## Conditions to revisit the decision

Before changing the execution topology, record at least a 7-day baseline for:

- task creation rate and peak concurrent tasks by task type;
- oldest pending-task age and pending-item backlog;
- task and item processing duration, failure, retry, and restart-recovery rates;
- the number of backend replicas and whether task execution must survive a
  rolling deployment without duplicate side effects.

Migration should be scheduled when observed backlog or task latency exceeds the
operator's service objective, when one process cannot provide the required
concurrency, or when more than one backend replica is required. A queue/worker
implementation must include a durable claim/lease, stale-lease recovery,
bounded concurrency, graceful shutdown, and idempotent processor effects before
horizontal execution is enabled.

## Interim operating rules

- Keep one backend execution process for task workloads.
- Treat `pending`, `running`, and `processing` rows as the recovery source of
  truth; do not manually invoke provider operations outside the task path.
- Use the existing administrator retry path for failed authorized-transfer
  tasks. Do not add automatic retries that could create duplicate shares.
- Re-evaluate this document after task traffic exists or before enabling a
  second backend replica.

## Measurement endpoint (2026-08-15)

The administrator-only `GET /api/tasks/operations/status` endpoint now exposes
the durable baseline needed for the next review. It accepts an optional
`window_days` query parameter from 1 through 90 (default: 7) and returns:

- task creation volume by type and status within the selected window;
- current pending/running task counts and pending/processing item counts;
- oldest pending task/item age in seconds;
- average and maximum duration for tasks with both `started_at` and
  `completed_at` in the selected window;
- the current process's running-task count and restart-recovery count.

The response contains aggregate operational data only. It does not return task
configuration, source URLs, share URLs, provider credentials, authorization
evidence, client IPs, or user content. The endpoint is intentionally a
measurement surface, not a queue claim or retry mechanism. Operators should
capture its seven-day results after task traffic exists and compare them with
the migration gates above before adding a second backend replica.
