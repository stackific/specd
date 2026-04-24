---
name: specd-new-tasks
description: Create tasks for a spec. Use when the user wants to break a specification into implementation tasks.
---

# Create tasks for a spec

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Task Markdown Convention

The `--body` you provide for each task becomes the task markdown body. It MUST follow this structure:

- Do NOT include a title in the body — the `--title` flag becomes the `# Title` heading automatically.
- Do NOT use `# Heading` (H1) in the body — only the title is H1.
- Use `##` for top-level sections. `###` through `######` are fine within sections.
- Include an `## Acceptance Criteria` section (must be H2) with checkbox items written as **claims** — each criterion MUST use must, should, is, or will language (e.g. "The handler must redirect…", "The response will include a JWT token").
- Task criteria use **checkbox syntax** (`- [ ] text`), NOT plain bullets.

### Example body structure

```markdown
## Overview

Implement the redirect handler that sends users to the OAuth2 consent screen.

## Requirements

- Build the authorization URL with correct scopes
- Include a cryptographically random state parameter

## Acceptance Criteria

- [ ] The handler must build the authorization URL with the correct client_id
- [ ] The state parameter should be at least 32 bytes of cryptographic randomness
- [ ] The scopes will be configurable via environment variables
```

## Steps

This is a four-step process. You MUST complete all four steps.

## Step 1: Find the spec

Determine whether the user provided a spec ID (e.g. "SPEC-1", "SPEC-42") or keywords.

### By ID

If the user provided a spec ID, run:

```sh
specd get-spec --id "<SPEC-ID>"
```

### By keywords

If the user provided search terms, run:

```sh
specd search --kind spec --query "<search terms>"
```

If the search returns multiple results, present them to the user and ask which spec to create tasks for. Do NOT proceed until the user confirms.

If no results are found, tell the user and stop.

Once you have the spec, read its title, summary, body, and acceptance criteria carefully. You will use these to decompose the spec into tasks.

## Step 2: Gap analysis — find uncovered acceptance criteria

The `get-spec` response includes:
- A `claims` array — the spec's own acceptance criteria (parsed from `## Acceptance Criteria` bullets in the spec body, written as must/should/is/will claims).
- A `tasks` array — each task has a `criteria` array with its acceptance criteria.

Compare them:

1. Collect every claim from the `claims` array — these are the spec's acceptance criteria.
2. Collect every criterion from all existing tasks (the `criteria` field on each task in the `tasks` array).
3. Match task criteria back to spec claims. A spec claim is **covered** if an existing task criterion addresses the same requirement (use semantic matching — wording does not need to be identical).
4. The **uncovered** spec claims are the ones no existing task addresses.

If all spec criteria are already covered, tell the user and stop — no new tasks needed.

If there are uncovered criteria, proceed to Step 3 using **only the uncovered criteria**. Do NOT duplicate work that existing tasks already cover.

## Step 3: Propose tasks and confirm

Analyze the **uncovered** acceptance criteria from Step 2, then propose new tasks that together cover them. For each proposed task, present:

1. **Title** — a concise, actionable title
2. **Summary** — a one-line description
3. **Key acceptance criteria** — the checkbox items you plan to include, mapped from the uncovered spec criteria. Each criterion MUST be written as a claim using must/should/is/will language

Also list which spec criteria are already covered by existing tasks, so the user has full visibility.

Present the full list to the user and ask for confirmation before creating any tasks. The user may want to:
- Add or remove tasks
- Modify titles or criteria
- Adjust scope

Do NOT create any tasks until the user explicitly confirms the plan.

## Step 4: Create the tasks

After the user confirms, create each task by running:

```sh
specd new-task --spec-id "<SPEC-ID>" --title "<title>" --summary "<one-line summary>" --body "<markdown body>"
```

Run this once for each confirmed task. The command outputs JSON with the task ID and path.

After creating all tasks, summarize what was created:

- Total number of tasks created
- Each task's ID and title
- The parent spec ID and title

## Examples

```sh
# Find spec by ID
specd get-spec --id "SPEC-3"

# Search for spec by keywords
specd search --kind spec --query "authentication"

# Create a task
specd new-task --spec-id "SPEC-3" --title "Implement OAuth redirect" --summary "Build the redirect handler for OAuth2 consent screen" --body "## Overview

Implement the redirect handler.

## Acceptance Criteria

- [ ] The handler must redirect to the OAuth2 consent screen
- [ ] The state parameter should be cryptographically random"
```
