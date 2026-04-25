---
name: specd-delete-task
description: Delete a task. Use when the user wants to remove a task that is no longer needed.
---

# Delete a task

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Important

Deleting a task is **destructive and irreversible**. It removes:
- The task markdown file from disk
- The task row from the database
- All task_criteria, task_links, task_dependencies, and citations referencing the task

The parent spec is NOT affected — only the task is deleted.

**Always confirm with the user before running the delete command.** Tell them what will be deleted and ask for explicit confirmation.

## Steps

## Step 1: Retrieve the task

If the user provides a task ID, retrieve it to show them what will be deleted:

```sh
specd get-task --id "<task-id>"
```

If they provide a title or description instead of an ID, search for it first:

```sh
specd search --kind task --query "<user's description>"
```

Show the user the task's title, summary, status, parent spec ID, and any acceptance criteria. Ask for explicit confirmation before proceeding.

## Step 2: Delete the task

After the user explicitly confirms deletion:

```sh
specd delete-task --id "<task-id>"
```

The command outputs JSON with:
- `id`: the deleted task ID
- `spec_id`: the parent spec ID
- `deleted`: boolean confirming deletion
- `path`: the file that was removed from disk

## Example

```sh
# Step 1: Review what will be deleted
specd get-task --id "TASK-3"

# Step 2: Delete after user confirms
specd delete-task --id "TASK-3"
```
