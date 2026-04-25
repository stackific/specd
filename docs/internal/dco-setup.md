# DCO Setup Guide for `specd`

## What is the DCO and why do we use it?

The **Developer Certificate of Origin (DCO)** is a lightweight alternative to a Contributor License Agreement. Instead of signing a separate legal document, every contributor certifies the origin of their code by adding a line to each commit message:

```
Signed-off-by: Your Name <you@example.com>
```

By adding that line, the contributor asserts (in short) that they wrote the code, or have the right to submit it under the project's license. Full text: https://developercertificate.org/

We chose DCO over a CLA because it is low-friction for contributors and widely recognized in Apache-2.0 projects (Linux kernel, Kubernetes, Docker, etc.).

**Non-negotiable rules for `specd`:**

1. Every commit merged into `main` must carry a `Signed-off-by` line.
2. The email in the `Signed-off-by` line must match the commit's author email.
3. Commits are blocked from merging if they fail the DCO check.

---

# Part 1: Configure the GitHub Org and Repo

Do this **once** per repo. Est. time: 10 minutes.

## 1.1 Install the DCO GitHub App

We use the official DCO App (https://github.com/apps/dco) rather than a custom GitHub Action. It is the de facto standard, surfaces clear fix-it instructions to contributors, and we do not maintain it.

**Steps:**

1. Sign in to GitHub as an owner of the `stackific` org.
2. Navigate to https://github.com/apps/dco.
3. Click **Install** (or **Configure** if already installed at the user level).
4. Select the `stackific` organization.
5. Choose **Only select repositories** and pick `specd`. (You can extend to more repos later.)
6. Click **Install**.

**Verify:** open any PR in `specd`. Within a minute or so you should see a `DCO` status check appear on the PR.

## 1.2 Configure branch protection on `main`

This makes the DCO check mandatory. Without this step, the app reports the status but merges are not actually blocked.

1. Go to `https://github.com/stackific/specd/settings/branches`.
2. Click **Add branch protection rule** (or edit the existing rule for `main`).
3. **Branch name pattern:** `main`
4. Enable the following:
   - [x] **Require a pull request before merging**
     - [x] Require approvals: **1** (adjust to project policy)
     - [x] Dismiss stale pull request approvals when new commits are pushed
     - [x] Require review from Code Owners
   - [x] **Require status checks to pass before merging**
     - [x] Require branches to be up to date before merging
     - In the search box, type `DCO` and select it as a required check. (It will only appear after at least one PR has triggered the app; if you do not see it yet, open a throwaway PR first, then come back.)
   - [x] **Require signed commits** - required. All commits merged to `main` must be cryptographically signed. See [commit-signing.md](commit-signing.md) for setup.
   - [x] **Require linear history** - recommended, keeps history clean.
   - [x] **Do not allow bypassing the above settings**
   - [x] **Restrict who can push to matching branches** - leave empty or restrict to maintainers group.
5. Save.

## 1.3 Add CODEOWNERS for legal-sensitive files

Create `.github/CODEOWNERS` in the `specd` repo:

```
# Default reviewers
*                   @stackific/maintainers

# Legal and licensing - require maintainer review
/LICENSE            @stackific/maintainers
/NOTICE             @stackific/maintainers
/TRADEMARKS.md      @stackific/maintainers
/.github/           @stackific/maintainers
```

Combined with **Require review from Code Owners** above, this prevents drive-by edits to licensing and governance files.

## 1.4 Document the PR template

Create `.github/pull_request_template.md`:

```markdown
## Summary

<!-- What does this PR do and why? -->

## Checklist

- [ ] All commits are signed off (DCO) and cryptographically signed. If not, run `git rebase --signoff --exec 'git commit --amend --no-edit -S' $(git merge-base HEAD origin/main) && git push --force-with-lease`
- [ ] Tests added or updated
- [ ] Docs updated if behavior or interfaces change
- [ ] I have read CONTRIBUTING.md
```

## 1.5 Configuration verification checklist

- [ ] DCO App installed on `stackific/specd`
- [ ] `main` has branch protection rule
- [ ] `DCO` is listed as a required status check
- [ ] CODEOWNERS file committed
- [ ] PR template committed
- [ ] A test PR with an unsigned commit is **blocked** from merging
- [ ] A test PR with a signed-off commit passes the DCO check

Once all boxes are checked, the repo is hardened. Hand off to developers.

---

# Part 2: Developer - One-Time Machine Setup

Do this **once** per machine. Est. time: 5 minutes.

After this, every commit you make to `specd` will automatically be signed off with the correct email, and you will never need to remember `-s` again.

## 2.1 Decide your work identity

For `specd` commits, use your **Stackific email** (e.g. `you@stackific.com`), not your personal email. This keeps your open source work attributed to Stackific and keeps the signoff consistent with an email tied to the company.

If you only ever work on Stackific repos from this machine, set it globally. If you mix personal and work, set it per-repo. Instructions for both below.

## 2.2 Add your work email to GitHub

1. Go to https://github.com/settings/emails.
2. Click **Add email address**, enter `you@stackific.com`, and verify via the confirmation email.
3. You do **not** need to make it your primary email. Verified is enough for GitHub to attribute commits to your account.
4. Under **Email privacy**, make sure **Block command line pushes that expose my email** is **off**. (If this is on, Git refuses to push commits with a real email. You want your real work email visible on commits.)

## 2.3 Configure Git

### Option A: Per-repo configuration (recommended if you use this machine for personal work too)

From inside your clone of `specd`:

```bash
cd /path/to/specd

# Identity used in commits
git config user.name  "Your Full Name"
git config user.email "you@stackific.com"

# Auto-add Signed-off-by to every commit in this repo
git config format.signoff true
```

### Option B: Global configuration (if this is a work-only machine)

```bash
git config --global user.name  "Your Full Name"
git config --global user.email "you@stackific.com"
git config --global format.signoff true
```

If you later contribute to a personal project from the same machine, override per-repo:

```bash
cd /path/to/personal-project
git config user.email "you@personal.com"
git config format.signoff false   # if you do not want signoff on personal work
```

## 2.4 Verify

From inside the `specd` repo:

```bash
# Confirm your identity
git config user.name
git config user.email
git config format.signoff

# Make a throwaway commit to verify
git commit --allow-empty -m "DCO setup test"
git log -1
```

The log output should show:

```
Author: Your Full Name <you@stackific.com>

    DCO setup test

    Signed-off-by: Your Full Name <you@stackific.com>
```

Key things to confirm:

1. **Author email** matches `you@stackific.com`.
2. **Signed-off-by email** matches the author email exactly (same case, same domain).
3. You did **not** have to pass `-s` to `git commit`.

Delete the test commit:

```bash
git reset --hard HEAD~1
```

You are done. Every future commit in `specd` will be signed off automatically.

## 2.5 Required: set up commit signing (GPG or SSH)

DCO signoff is a text line — it is not cryptographic. This project **also requires** cryptographic commit signing. All commits merged to `main` must be signed, enforced via branch protection.

See [commit-signing.md](commit-signing.md) for full setup instructions.

Commit signing is separate from DCO and does not replace the `Signed-off-by` line. Both are required.

---

# Part 3: Troubleshooting and Common Scenarios

## 3.1 "My PR is failing the DCO check"

The check lists which commits are missing or malformed. Most common cases below.

### Case A: Forgot to sign off one or more commits

From your PR branch:

```bash
git fetch origin
git rebase --signoff $(git merge-base HEAD origin/main)
git push --force-with-lease
```

This adds `Signed-off-by` to every commit in the PR that lacks one, without rebasing onto new changes from `main`. Commits that already have a valid signoff are left untouched.

### Case B: Only the latest commit is unsigned

```bash
git commit --amend --signoff --no-edit
git push --force-with-lease
```

### Case C: The Signed-off-by email does not match the author email

This happens if you changed your Git `user.email` partway through the PR, or you have different emails at the author vs committer level.

Rewrite all commits in the PR with a consistent author and signoff:

```bash
git rebase -i $(git merge-base HEAD origin/main) \
  --exec 'git commit --amend --author="Your Name <you@stackific.com>" --no-edit -s'
git push --force-with-lease
```

### Case D: The PR contains commits authored by someone else

`git rebase --signoff` would sign those commits off **as you**, which is not what DCO is for. You are not allowed to certify code you did not write.

Options:

1. Ask the original author to sign off their own commits and push.
2. If the author is unreachable, drop their commits or reimplement the changes yourself and sign off as you.

## 3.2 "My commits show up on GitHub as an unknown author with no avatar"

The commit email is valid but not attached to your GitHub account. Fix: add that email at https://github.com/settings/emails and verify it. No rewrite needed; GitHub re-attributes historical commits automatically once the email is verified.

## 3.3 "I already pushed 20 commits with my personal email"

Rewrite the PR branch to use the work email consistently:

```bash
git rebase -i $(git merge-base HEAD origin/main) \
  --exec 'git commit --amend --author="Your Name <you@stackific.com>" --no-edit -s'
git push --force-with-lease
```

This changes both the author line and re-adds a matching signoff. DCO will now pass.

Do this only on your own PR branches. Never rewrite history on `main` or any shared branch.

## 3.4 "Force-push scares me"

Use `--force-with-lease` instead of `--force`. It refuses to overwrite the remote if someone else has pushed to the branch since you last fetched. This is safe on your own PR branches.

Never force-push to `main`. Branch protection should prevent it, but do not rely on that alone.

## 3.5 "I use the GitHub web editor and it does not sign commits off"

The web editor uses your GitHub account's primary email and does **not** add a signoff line. Workarounds:

1. Avoid the web editor for `specd` changes. Commit from the command line.
2. Or: immediately after editing in the web UI, fetch the PR branch locally, rebase with signoff, and push.

## 3.6 "I use VS Code / JetBrains / another GUI. Do I need to do anything extra?"

No. These tools respect `git config format.signoff true` and your `user.email` setting. Once step 2.3 is done, GUI commits will be signed off automatically.

Exception: GitHub Desktop historically has not respected `format.signoff`. If you use GitHub Desktop, commit from the CLI or manually add the signoff line in the commit message box.

## 3.7 "I have commits authored by multiple co-authors (pair programming)"

Use Git's `Co-authored-by` trailer in addition to your own signoff:

```
Implement widget factory

Signed-off-by: Ahmed Smith <ahmed@stackific.com>
Co-authored-by: Bob Jones <bob@stackific.com>
Signed-off-by: Bob Jones <bob@stackific.com>
```

Each author who contributed must have their own signoff line. `Co-authored-by` alone is not enough for DCO.

---

# Part 4: Quick Reference

## Developer cheat sheet

| Situation | Command |
|---|---|
| One-time setup per repo | `git config user.email "you@stackific.com" && git config format.signoff true` |
| Normal commit | `git commit -m "message"` (signoff is automatic) |
| Forgot signoff on last commit | `git commit --amend --signoff --no-edit && git push --force-with-lease` |
| Forgot signoff on multiple commits in PR | `git rebase --signoff $(git merge-base HEAD origin/main) && git push --force-with-lease` |
| Used wrong email across whole PR | `git rebase -i $(git merge-base HEAD origin/main) --exec 'git commit --amend --author="Your Name <you@stackific.com>" --no-edit -s' && git push --force-with-lease` |
| Verify a commit has signoff | `git log -1 --format=%B` |

## Configuration cheat sheet

| Task | Where |
|---|---|
| Install DCO App | https://github.com/apps/dco |
| Configure branch protection | `https://github.com/stackific/specd/settings/branches` |
| Required status check name | `DCO` |
| CODEOWNERS file path | `.github/CODEOWNERS` |
| PR template path | `.github/pull_request_template.md` |

## Anatomy of a valid DCO-compliant commit

```
Fix parser crash on empty input

The parser previously panicked when passed an empty string.
Guard against nil input and return a typed error instead.

Signed-off-by: Jane Khan <jane@stackific.com>
```

- Subject line: imperative mood, under 72 chars.
- Blank line.
- Body: what and why (not how).
- Blank line.
- `Signed-off-by:` line with real name and an email matching `Author`.

---

# Part 5: FAQ

**Q: Do I need a separate CLA?**
No. DCO replaces the CLA for this project.

**Q: Can I use a pseudonym?**
No. DCO requires a real legal name. Use the name you would sign a contract with.

**Q: Can I use an `@users.noreply.github.com` email?**
Technically DCO will pass as long as author and signoff match, but for `specd` we require a real verifiable email. Use your `@stackific.com` address.

**Q: What if I contribute to `specd` from my personal account outside work hours?**
Discuss with your manager. Generally, if the contribution is to a repo owned by `stackific`, use the `@stackific.com` email and account attribution.

**Q: What happens to contributions from external (non-Stackific) contributors?**
Same rules apply, with their own identity. The DCO App enforces this uniformly. Our `CONTRIBUTING.md` explains the DCO requirement to external contributors.

**Q: Can we grandfather in old unsigned commits from before we adopted DCO?**
The DCO App checks only commits in the PR diff, not full history. Commits already merged to `main` before DCO was adopted are not rechecked. New commits going forward must be signed off.

**Q: What if the DCO App goes down?**
The check becomes pending and merges are blocked (by design). If this persists, sysadmin can temporarily remove `DCO` from required status checks, merge critical fixes, and restore the requirement after. This should be rare.

---

# Part 6: Related Documents

- `.github/CONTRIBUTING.md` - contributor-facing guide, contains DCO and commit signing explanation for outside contributors.
- `.github/CODE_OF_CONDUCT.md` - community standards.
- `.github/CODEOWNERS` - reviewer assignments.
- `.github/pull_request_template.md` - PR checklist.
- `docs/internal/commit-signing.md` - cryptographic commit signing setup (SSH/GPG).
- `LICENSE` - Apache 2.0 text.
- `NOTICE` - attribution notice.
- `TRADEMARKS.md` - trademark policy for `specd` and `Stackific`.

---

*Document owner: Stackific Engineering. Last reviewed: 2026-04-21. For questions, contact info@stackific.com.*