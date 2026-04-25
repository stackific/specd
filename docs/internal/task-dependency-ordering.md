# Task Dependency Ordering

How to determine the execution sequence of tasks for a spec given the `depends_on` and `linked_tasks` relationships.

## Current Data Model

Each task can declare:
- **`depends_on`** — a list of task IDs that **block** this task. Stored as directed edges in `task_dependencies(blocker_task, blocked_task)`. Meaning: "I cannot start until these tasks are done."
- **`linked_tasks`** — bidirectional, undirected relationships in `task_links`. These indicate relatedness, not ordering.

Only `depends_on` implies execution order. `linked_tasks` is informational.

## The Graph

Tasks + `depends_on` edges form a **directed acyclic graph (DAG)**. Each edge `(A, B)` means "A blocks B" — A must complete before B can start.

```
TASK-1 ──→ TASK-3 ──→ TASK-5
TASK-2 ──→ TASK-3
TASK-4 (no dependencies — independent)
```

In this example:
- TASK-1 and TASK-2 can run in parallel (no deps)
- TASK-3 waits for both TASK-1 and TASK-2
- TASK-5 waits for TASK-3
- TASK-4 can run anytime

## Deriving Execution Order

### Approach 1: Topological Sort (Full Ordering)

A topological sort of the DAG produces a valid execution sequence. Standard Kahn's algorithm:

1. Compute in-degree (number of blockers) for each task
2. Start with all tasks that have in-degree 0 (no blockers) — these form the first "wave"
3. Remove those tasks from the graph, decrement in-degree of tasks they block
4. Repeat until all tasks are placed

This gives a **linear** ordering, but multiple valid orderings exist. Tasks within the same "wave" (same topological level) can run in parallel.

### Approach 2: Critical Path (Parallelism-Aware)

Group tasks into **levels** based on the longest path from any root:

- **Level 0**: Tasks with no dependencies (can start immediately)
- **Level N**: Tasks whose latest-finishing blocker is at level N-1

This naturally identifies parallelism: all tasks at the same level can be worked on concurrently.

### Approach 3: AI-Driven Ordering (What We Already Have)

The `/specd-new-tasks` skill already proposes tasks in a logical order when decomposing a spec. The AI can:

1. Look at the `depends_on` edges from `get-spec` or `get-task` responses
2. Build the dependency graph mentally
3. Suggest which tasks to work on next based on what's already done

This avoids needing a dedicated topological sort command — the AI interprets the graph.

## Implementation Options

### Option A: Query-Time Ordering (No New Command)

Add a SQL query that returns tasks in dependency order:

```sql
-- Recursive CTE to compute dependency depth
WITH RECURSIVE dep_depth(task_id, depth) AS (
  -- Base: tasks with no blockers start at depth 0
  SELECT t.id, 0
  FROM tasks t
  WHERE t.spec_id = ?
  AND NOT EXISTS (
    SELECT 1 FROM task_dependencies td WHERE td.blocked_task = t.id
  )
  UNION ALL
  -- Recursive: depth = max(blocker depths) + 1
  SELECT td.blocked_task, dd.depth + 1
  FROM task_dependencies td
  JOIN dep_depth dd ON dd.task_id = td.blocker_task
)
SELECT task_id, MAX(depth) as level
FROM dep_depth
GROUP BY task_id
ORDER BY level, task_id
```

This could be exposed via `specd list-tasks --spec-id SPEC-1 --order deps` or included in the `get-spec` response as a `task_order` field.

### Option B: Add Dependency Info to get-spec Response

Extend `GetSpecTask` to include `depends_on` for each task. The AI skill can then build the graph from the response without extra queries. This is the simplest approach — no new commands needed.

### Option C: Dedicated `specd task-order` Command

A command that outputs the DAG as ordered levels:

```json
{
  "spec_id": "SPEC-1",
  "levels": [
    {"level": 0, "tasks": ["TASK-1", "TASK-2", "TASK-4"]},
    {"level": 1, "tasks": ["TASK-3"]},
    {"level": 2, "tasks": ["TASK-5"]}
  ]
}
```

### Recommendation

**Option B is the simplest and most useful.** Add `depends_on` to the `GetSpecTask` struct in the `get-spec` response. The AI already has the full task list and criteria — adding dependency edges lets it compute ordering without a new command. If a dedicated command is needed later, Option C is straightforward to implement on top.

## Cycle Detection

The `depends_on` graph must be acyclic. If a user manually edits frontmatter and creates a cycle (A → B → C → A), the topological sort would fail. Options:

1. **Detect at sync time** — reject the cyclic task with a warning in logs
2. **Detect at `new-task` / `update-task` time** — return an error before writing
3. **Detect at query time** — the recursive CTE would loop; use a depth limit

Option 2 is safest — fail fast when the dependency is created.

## Status-Aware Ordering

Tasks have a `status` field. A task with status `done` satisfies its dependency — blocked tasks can proceed. The ordering query can filter:

- "What can I work on next?" → tasks at level 0 (all blockers are `done`) that are not yet `done`
- "What's blocking progress?" → tasks whose blockers are not `done`
