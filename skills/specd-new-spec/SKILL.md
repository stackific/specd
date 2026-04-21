---
name: specd-new-spec
description: Create a new spec from a description. Use when the user wants to add a specification, requirement, or feature description to the project.
---

# Create a new spec

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Steps

This is a two-step process. You MUST complete both steps.

## Step 1: Create the spec

From the user's description, determine:
- **title**: a concise title for the spec
- **summary**: a one-line summary
- **body**: the full spec content in markdown

Then run:

```sh
specd new-spec --title "<title>" --summary "<one-line summary>" --body "<markdown body>"
```

The command outputs JSON with:
- `id`: the assigned spec ID (e.g. "SPEC-1")
- `path`: where the spec.md file was written
- `default_type`: the default spec type assigned
- `available_types`: all spec types the user configured
- `related_specs`: top matching existing specs (with id and summary)
- `related_kb_chunks`: top matching KB chunks (with chunk_id, doc_id, preview)

## Step 2: Set the type and links

From the JSON response, decide:
1. **Spec type**: which of `available_types` best fits this spec
2. **Linked specs**: which `related_specs` (if any) are genuinely related — use the IDs
3. **Linked KB chunks**: which `related_kb_chunks` (if any) are relevant — use the chunk_ids

Then run:

```sh
specd update-spec --id "<spec-id>" --type "<chosen-type>" --link-specs "<SPEC-X,SPEC-Y>" --link-kb-chunks "<1,2,3>"
```

Omit `--link-specs` or `--link-kb-chunks` if none are relevant. Always set `--type`.

## Example

```sh
# Step 1
specd new-spec --title "User Authentication" --summary "OAuth2 login flow for web app" --body "## Overview\n\nImplement OAuth2..."

# Step 2 (using the returned SPEC-1 id and choosing from the response)
specd update-spec --id "SPEC-1" --type "functional" --link-specs "SPEC-3,SPEC-5"
```
