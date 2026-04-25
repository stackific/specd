---
name: specd-check-task
description: Toggle task acceptance criteria. Use when the user wants to check or uncheck acceptance criteria on a task.
---

# Toggle task acceptance criteria

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

If the search returns multiple results, present them to the user and ask which task to update. Do NOT proceed until the user confirms.

If no results are found, tell the user and stop.

Once you have the task, review its `criteria` array. Each criterion has a `position`, `text`, and `checked` state (0 = unchecked, 1 = checked).

## Step 2: Determine which criteria to toggle

Based on the user's request, identify which criteria to check or uncheck. The user may say things like:

- "Check criterion 1" → check position 1
- "Mark the JWT token criterion as done" → find the matching criterion by text, use its position
- "Uncheck criterion 2" → uncheck position 2
- "Check all criteria" → check all positions
- "The credentials validation is done" → find the matching criterion by text, check it

Present the changes you plan to make and ask the user to confirm:

- List each criterion with its current state and the new state
- Show the position number and text for clarity

Do NOT make any changes until the user confirms.

## Step 3: Apply the changes

Run the update command with the appropriate flags:

```sh
specd update-task --id "<TASK-ID>" --check "<positions>" --uncheck "<positions>"
```

- `--check` takes a comma-separated list of 1-based criterion positions to mark as checked
- `--uncheck` takes a comma-separated list of 1-based criterion positions to mark as unchecked
- Omit either flag if no changes are needed for that direction

The response includes:
- `id`: the task ID
- `spec_id`: the parent spec ID
- `status`: the current task status
- `criteria`: the full list of criteria with their updated checked state

After the update, summarize what changed:
- Which criteria were checked or unchecked
- The task ID and title

## Examples

```sh
# Get a task to see its criteria
specd get-task --id "TASK-3"

# Check criteria at positions 1 and 3
specd update-task --id "TASK-3" --check "1,3"

# Uncheck criterion at position 2
specd update-task --id "TASK-3" --uncheck "2"

# Check some and uncheck others in one call
specd update-task --id "TASK-3" --check "1,3" --uncheck "2"
```
