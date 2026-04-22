---
name: specd-delete-spec
description: Delete a spec and all its tasks. Use when the user wants to remove a specification that is no longer needed.
---

# Delete a spec

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Important

Deleting a spec is **destructive and irreversible**. It removes:
- The spec markdown file and its directory from disk
- The spec row from the database
- All tasks linked to the spec (via ON DELETE CASCADE)
- All spec_links, task_criteria, task_links, task_dependencies, and citations referencing the spec

**Always confirm with the user before running the delete command.** Tell them what will be deleted and ask for explicit confirmation.

## Steps

## Step 1: Retrieve the spec

If the user provides a spec ID, retrieve it to show them what will be deleted:

```sh
specd get-spec --id "<spec-id>"
```

If they provide a title or description instead of an ID, search for it first:

```sh
specd search --kind spec --query "<user's description>"
```

Show the user the spec's title, summary, type, and any linked specs or tasks. Ask for explicit confirmation before proceeding.

## Step 2: Delete the spec

After the user explicitly confirms deletion:

```sh
specd delete-spec --id "<spec-id>"
```

The command outputs JSON with:
- `id`: the deleted spec ID
- `deleted`: boolean confirming deletion
- `path`: the directory that was removed from disk

## Example

```sh
# Step 1: Review what will be deleted
specd get-spec --id "SPEC-3"

# Step 2: Delete after user confirms
specd delete-spec --id "SPEC-3"
```
