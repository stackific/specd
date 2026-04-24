---
name: specd-find-task
description: Find a task by ID or search by keywords. Use when the user asks to look up, find, or search for a task.
---

# Find a task

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## How to find a task

Determine whether the user provided a task ID (e.g. "TASK-1", "TASK-42") or keywords.

### By ID

If the user provided a task ID, run:

```sh
specd get-task --id "<TASK-ID>"
```

This returns full JSON with all fields including title, summary, body, status, spec_id, criteria, linked tasks, dependencies, and timestamps.

### By keywords

If the user provided search terms, run:

```sh
specd search --kind task --query "<search terms>"
```

This returns JSON with a `tasks` array of matching results ranked by relevance. Each result includes id, title, summary, score, and match_type.

You can also pass `--limit <N>` to control the number of results.

## Output to user

After receiving the JSON response:

- If a single task was found (by ID), tell the user the task ID, title, status, parent spec ID, and summary. If it has acceptance criteria, list them with their checked state.
- If searching by keywords, list the matching tasks with their IDs, titles, and statuses, ordered by relevance.
- If no results were found, tell the user no tasks matched their query.

## Examples

```sh
# By ID
specd get-task --id "TASK-3"

# By keywords
specd search --kind task --query "database migration"

# By keywords with limit
specd search --kind task --query "authentication" --limit 3
```
