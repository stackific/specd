---
name: specd-new-spec
description: Create a new spec from a description. Use when the user wants to add a specification, requirement, or feature description to the project.
---

# Create a new spec

## Prerequisite

Before running this skill, check if a `.specd.json` file exists in the current directory. If it does not, stop and tell the user:

"Please run `specd init` in your terminal first to initialize the project, then try again."

Do NOT suggest alternative ways to run the command. Do NOT mention shell prefixes, prompt shortcuts, or inline execution. Just tell them to run the command in their terminal.

## Spec Markdown Convention

The `--body` you provide becomes the spec markdown body. It MUST follow this structure:

- Do NOT include a title in the body — the `--title` flag becomes the `# Title` heading automatically.
- Do NOT use `# Heading` (H1) in the body — only the title is H1.
- Use `##` for top-level sections. `###` through `######` are fine within sections.
- Include an `## Acceptance Criteria` section (must be H2) with bullet items written as claims using must, should, may, or might language.

### Example body structure

```markdown
## Overview

Users need to authenticate via OAuth2 providers.

## Requirements

- Support Google and GitHub OAuth2 providers
- Issue JWT tokens after successful authentication

## Acceptance Criteria

- The system must redirect users to the OAuth2 provider's consent screen
- The system should create new user records on first login
- Users may choose to stay logged in via a remember-me option
```

## Steps

This is a three-step process. You MUST complete all three steps.

## Step 1: Create the spec

From the user's description, determine:
- **title**: a concise title for the spec (becomes the H1 heading)
- **summary**: a one-line summary
- **body**: the full spec content following the convention above

Then run:

```sh
specd new-spec --title "<title>" --summary "<one-line summary>" --body "<markdown body>"
```

The command outputs JSON with the spec ID, path, available types, and related content.

## Step 2: Set the type and links

From the step 1 JSON response, decide:
1. **Spec type**: which of `available_types` best fits this spec
2. **Linked specs**: which `related_specs` (if any) are genuinely related
3. **Linked KB chunks**: which `related_kb_chunks` (if any) are relevant

Then run:

```sh
specd update-spec --id "<spec-id>" --type "<chosen-type>" --link-specs "<SPEC-X,SPEC-Y>" --link-kb-chunks "<1,2,3>"
```

Omit `--link-specs` or `--link-kb-chunks` if none are relevant. Always set `--type`.

## Step 3: Check for contradictions

For each acceptance criteria claim in the spec you just created, search for potentially conflicting claims in other specs:

```sh
specd search-claims --query "<claim text>" --exclude "<spec-id>"
```

Run this once for each claim in the `## Acceptance Criteria` section. The command returns matching claims from other specs with their spec IDs and titles.

Review the returned claims. For each match, evaluate whether it genuinely contradicts the new claim. If you find contradictions, report them to the user:

- The spec ID and title of the conflicting spec
- The conflicting claim from the other spec
- The claim from the new spec that it conflicts with

Do NOT take any action to resolve the conflicts. Just report them and exit.

If no contradictions are found, tell the user the spec was created cleanly with no conflicts detected.
