---
name: specd-find-spec
description: Find a spec by ID or search by keywords. Use when the user asks to look up, find, or search for a specification.
---

# Find a spec

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## How to find a spec

Determine whether the user provided a spec ID (e.g. "SPEC-1", "SPEC-42") or keywords.

### By ID

If the user provided a spec ID, run:

```sh
specd get-spec --id "<SPEC-ID>"
```

This returns full JSON with all fields including title, summary, body, type, linked specs, and timestamps.

### By keywords

If the user provided search terms, run:

```sh
specd search --kind spec --query "<search terms>"
```

This returns JSON with a `specs` array of matching results ranked by relevance. Each result includes id, title, summary, score, and match_type.

You can also pass `--limit <N>` to control the number of results.

## Output to user

After receiving the JSON response:

- If a single spec was found (by ID), tell the user the spec ID, title, type, and summary.
- If searching by keywords, list the matching specs with their IDs and titles, ordered by relevance.
- If no results were found, tell the user no specs matched their query.

## Examples

```sh
# By ID
specd get-spec --id "SPEC-3"

# By keywords
specd search --kind spec --query "authentication login"

# By keywords with limit
specd search --kind spec --query "rate limiting" --limit 3
```
