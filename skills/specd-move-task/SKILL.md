---
name: specd-move-task
description: Move a task to a different stage. Use when the user wants to change a task's status (e.g. backlog to todo, in progress, done).
---

# Move a task to a different stage

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Steps

This is a three-step process. You MUST complete all three steps.

## Step 1: Find the task

Determine whether the user provided a task ID (e.g. "TASK-1", "TASK-42") or keywords.

### By ID

If the user provided a task ID, run:

```sh
specd get-task --id "<TASK-ID>"
```

### By keywords

If the user provided search terms, run:

```sh
specd search --kind task --query "<search terms>"
```

If the search returns multiple results, present them to the user and ask which task to move. Do NOT proceed until the user confirms.

If no results are found, tell the user and stop.

Once you have the task, note its current `status`.

## Step 2: Determine the target stage

Read the project config to see available task stages:

```sh
cat .specd.json
```

The `task_stages` array lists all valid stages (e.g. `["backlog", "todo", "in_progress", "done", "blocked", "pending_verification", "cancelled", "wont_fix"]`).

Based on the user's request, determine the target stage. The user may say things like:

- "Move TASK-1 to done" → target stage is `done`
- "Start working on the login task" → target stage is `in_progress`
- "Block TASK-3" → target stage is `blocked`
- "Send TASK-2 back to backlog" → target stage is `backlog`

Confirm the move with the user:
- Show the task ID, title, and current stage
- Show the target stage
- Ask for confirmation

Do NOT make any changes until the user confirms.

## Step 3: Apply the change

Run the update command:

```sh
specd update-task --id "<TASK-ID>" --status "<target-stage>"
```

The `--status` value must exactly match one of the stages from `task_stages` in `.specd.json` (underscore-separated slug format, e.g. `in_progress`, not "In Progress").

The response includes:
- `id`: the task ID
- `spec_id`: the parent spec ID
- `status`: the new status after the move
- `criteria`: the full list of criteria with their checked state

After the update, summarize what changed:
- The task ID and title
- The previous stage and the new stage

## Examples

```sh
# Get a task to see its current status
specd get-task --id "TASK-5"

# Move to in_progress
specd update-task --id "TASK-5" --status "in_progress"

# Move to done
specd update-task --id "TASK-5" --status "done"

# Move back to backlog
specd update-task --id "TASK-5" --status "backlog"
```
