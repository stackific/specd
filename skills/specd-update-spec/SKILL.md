---
name: specd-update-spec
description: Update an existing spec — change its type, add or remove linked specs, and add or remove KB chunk citations. Use when the user wants to modify a specification's metadata or references.
---

# Update a spec

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Steps

## Step 1: Retrieve the spec

Get the spec by ID so you can see its current state:

```sh
specd get-spec --id "<spec-id>"
```

The response includes:
- `id`, `title`, `type`, `summary`, `body`
- `linked_specs`: currently linked spec IDs
- `content_hash`, `created_at`, `updated_at`

Review the spec and determine what changes the user wants.

## Step 2: Update the spec

Run the update command with only the flags that need to change:

```sh
specd update-spec --id "<spec-id>" \
  --type "<new-type>" \
  --link-specs "<SPEC-X,SPEC-Y>" \
  --unlink-specs "<SPEC-Z>" \
  --link-kb-chunks "<1,2>" \
  --unlink-kb-chunks "<3>"
```

All flags except `--id` are optional — omit any that don't apply.

| Flag | Description |
|------|-------------|
| `--type` | Set the spec type (must be one of the configured types) |
| `--link-specs` | Comma-separated spec IDs to add as related |
| `--unlink-specs` | Comma-separated spec IDs to remove from related |
| `--link-kb-chunks` | Comma-separated KB chunk IDs to cite |
| `--unlink-kb-chunks` | Comma-separated KB chunk IDs to remove from citations |

The response includes the final state after all changes:
- `id`: the spec ID
- `type`: the current type after update
- `linked_specs`: array of `{id, title, summary}` for all currently linked specs
- `linked_kb_chunks`: array of `{chunk_id, doc_id, preview}` for all currently cited KB chunks

The spec.md file on disk is automatically rewritten to reflect the changes.

## Step 3: Check for contradictions

After the update, if the spec has acceptance criteria claims, check for contradictions with other specs. For each claim in the spec's `## Acceptance Criteria` section, run:

```sh
specd search-claims --query "<claim text>" --exclude "<spec-id>"
```

Review the returned claims. For each match, evaluate whether it genuinely contradicts the claim in this spec. If you find contradictions, report them to the user:

- The spec ID and title of the conflicting spec
- The conflicting claim from the other spec
- The claim from this spec that it conflicts with

Do NOT take any action to resolve the conflicts. Just report them and exit.

If no contradictions are found, tell the user the spec was updated cleanly with no conflicts detected.

## Example

```sh
# Step 1: Review the spec
specd get-spec --id "SPEC-3"

# Step 2: Change type, add a link, remove another
specd update-spec --id "SPEC-3" --type "functional" --link-specs "SPEC-7" --unlink-specs "SPEC-1"

# Step 3: Check for contradictions in each claim
specd search-claims --query "The system must validate tokens" --exclude "SPEC-3"
specd search-claims --query "Sessions should expire after 15 minutes" --exclude "SPEC-3"
```
