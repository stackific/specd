# Commit Signing Guide for `specd`

## What is commit signing and why do we use it?

A Git commit's author field is just a string. Anyone can set `user.name` and `user.email` to any value and commit as "Linus Torvalds <torvalds@kernel.org>". Without signing, there is no way to tell a real commit from an impersonation.

Cryptographic commit signing attaches a signature to each commit, produced by a private key only the author holds. GitHub verifies the signature against a public key registered to the author's account and displays a **Verified** badge. Unsigned or invalid commits show **Unverified** or nothing.

**Why this matters for `specd`:**

1. We publish binaries. If a malicious commit slipped into `main`, anyone running the binary is exposed. Signatures make impersonation detectable.
2. We accept external contributions. Signed commits give maintainers confidence that a PR from `ahmed-stackific` really came from Ahmed.
3. Supply-chain attestations (SLSA, provenance) build on signed commits. Adopting signing now means we are ready when customers ask.

**Non-negotiable rules for `specd`:**

1. Every commit merged to `main` must be cryptographically signed.
2. The signing key must be registered to the GitHub account that authored the commit.
3. Unsigned commits are blocked from merging via branch protection.

## SSH or GPG?

GitHub supports signing with either SSH keys or GPG keys. Both produce a **Verified** badge and satisfy branch protection. Pick one.

| | SSH signing | GPG signing |
|---|---|---|
| Setup difficulty | Easy. Reuses your existing SSH key. | Harder. New key, new tooling. |
| Key management | One key for auth and signing. | Separate keys. |
| Requires Git version | 2.34+ | Any |
| Works offline | Yes | Yes |
| Ecosystem familiarity | Newer (since 2021). | Long-established. |
| Revocation story | Simple: remove key from GitHub. | GPG revocation certificates. |
| Smartcard / YubiKey | Yes (OpenSSH + hardware key) | Yes (OpenPGP smartcards) |

**Stackific standard: SSH signing.** It is the simpler path, reuses the key you already use to push, and is the modern default. This guide documents SSH signing as the primary path and includes GPG as an appendix for developers who already use GPG or prefer it.

---

# Part 1: As an org. admin, configure the GitHub org and repo

Do this **once** per repo.

## 1.1 Enable "Require signed commits" on `main`

1. Go to `https://github.com/stackific/specd/settings/branches`.
2. Edit the existing branch protection rule for `main` (created during DCO setup).
3. Check **Require signed commits**.
4. Save.

**Effect:** PRs containing unsigned commits cannot be merged. GitHub shows a red cross next to the offending commits in the PR's **Commits** tab.

## 1.2 Verification checklist

- [ ] `Require signed commits` enabled on `main` branch protection
- [ ] Test PR with an unsigned commit is blocked
- [ ] Test PR with a signed commit shows **Verified** on each commit and merges successfully

---

# Part 2: Developer - One-Time Machine Setup (SSH signing)

Do this **once** per machine. Est. time: 5 minutes.

You already push to GitHub over SSH, so you already have an SSH key. We are going to tell Git to use that same key to sign commits.

Requires Git **2.34 or newer**. Check with `git --version`. If older, upgrade first (`brew upgrade git`, `apt install git`, etc.).

## 2.1 Locate your SSH key

Default location is `~/.ssh/id_ed25519.pub` (or `~/.ssh/id_rsa.pub`). List your public keys:

```bash
ls -1 ~/.ssh/*.pub
```

If you do not have one, create an Ed25519 key:

```bash
ssh-keygen -t ed25519 -C "you@stackific.com"
```

Accept the default path. Set a passphrase (strongly recommended; your OS keychain will cache it).

## 2.2 Register the key as a signing key on GitHub

A single SSH key can be registered twice: once for authentication (pushing) and once for signing. GitHub treats them as separate entries.

1. Copy your public key:

   ```bash
   # macOS
   pbcopy < ~/.ssh/id_ed25519.pub

   # Linux with xclip
   xclip -selection clipboard < ~/.ssh/id_ed25519.pub

   # Or just print and copy manually
   cat ~/.ssh/id_ed25519.pub
   ```

2. Go to https://github.com/settings/keys.
3. Click **New SSH key**.
4. **Title:** something like `laptop-2026-signing`.
5. **Key type:** change from "Authentication Key" to **Signing Key**.
6. Paste the key. Save.

If you also want this key for authentication (pushing), repeat the steps with **Key type: Authentication Key**. The same public key text goes in both entries.

## 2.3 Configure Git to sign with SSH

Run these in your `specd` clone (or globally with `--global` if this is a work-only machine):

```bash
# Use SSH (not GPG) as the signing format
git config gpg.format ssh

# Point Git at your SSH public key for signing
git config user.signingkey ~/.ssh/id_ed25519.pub

# Sign every commit automatically
git config commit.gpgsign true

# Sign every tag automatically (recommended)
git config tag.gpgsign true
```

For global configuration (work-only machine), add `--global` to each command.

## 2.4 Tell Git how to verify other people's signatures (optional but recommended)

Without an allowed-signers file, `git log --show-signature` will complain it cannot verify signatures locally, even though GitHub can. To silence this and enable local verification:

```bash
# Create an allowed-signers file
mkdir -p ~/.ssh
touch ~/.ssh/allowed_signers

# Add your own key to it
echo "you@stackific.com $(cat ~/.ssh/id_ed25519.pub)" >> ~/.ssh/allowed_signers

# Tell Git where it is
git config --global gpg.ssh.allowedSignersFile ~/.ssh/allowed_signers
```

Add teammates' public keys to the same file as you collaborate with them, one per line. This step is cosmetic for `specd` (GitHub does the real verification) but useful for local auditing.

## 2.5 Verify

From inside the `specd` repo:

```bash
# Confirm config
git config gpg.format          # should print: ssh
git config user.signingkey     # should print path to your .pub file
git config commit.gpgsign      # should print: true

# Make a throwaway signed commit
git commit --allow-empty -m "Signing setup test"

# Inspect it
git log -1 --show-signature
```

Expected output includes a line starting with `Good "git" signature` (or similar, depending on whether `allowed_signers` is set).

Push the commit to a throwaway branch and check GitHub:

```bash
git checkout -b signing-test
git push -u origin signing-test
```

On GitHub, navigate to the branch's latest commit. It should show a green **Verified** badge. If it shows **Unverified**, see troubleshooting below.

Clean up:

```bash
git checkout main
git branch -D signing-test
git push origin --delete signing-test
```

You are done. Every future commit will be signed automatically. Combined with the DCO `format.signoff` setting, your commits will satisfy both requirements with no per-commit flags.

## 2.6 Combined config (DCO + signing)

For convenience, here is the full recommended Git config for `specd` contributors, applied per-repo:

```bash
cd /path/to/specd

# Identity (DCO requirement: author email must match Signed-off-by)
git config user.name  "Your Full Name"
git config user.email "you@stackific.com"

# DCO auto-signoff
git config format.signoff true

# Cryptographic signing
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519.pub
git config commit.gpgsign true
git config tag.gpgsign true
```

Six commands, one time, never think about it again.

## 2.7 Multiple emails / keys (personal + work)

If you use a personal email for open-source and a `@stackific.com` email for work, generate a separate key per email and configure per-repo:

### Generate both keys

```bash
ssh-keygen -t ed25519 -C "you@personal.com" -f ~/.ssh/id_ed25519_personal
ssh-keygen -t ed25519 -C "you@stackific.com" -f ~/.ssh/id_ed25519_work
```

### Register both on GitHub as signing keys

```bash
gh ssh-key add ~/.ssh/id_ed25519_personal.pub --title "personal signing" --type signing
gh ssh-key add ~/.ssh/id_ed25519_work.pub --title "work signing" --type signing
```

### Set your personal email as the global default

```bash
git config --global user.name "Your Full Name"
git config --global user.email "you@personal.com"
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519_personal.pub
git config --global commit.gpgsign true
git config --global tag.gpgsign true
```

### Override per-repo for Stackific work

Inside each Stackific repo (e.g. `specd`), set the work identity:

```bash
cd /path/to/specd
git config user.email "you@stackific.com"
git config user.signingkey ~/.ssh/id_ed25519_work.pub
```

That's it. `user.name`, `gpg.format`, `commit.gpgsign`, and `tag.gpgsign` inherit from global. Only `user.email` and `user.signingkey` differ per repo.

### How it works

Git resolves config in order: **repo > global > system**. Per-repo settings in `.git/config` override `~/.gitconfig`. So:

- In `specd`: commits use `you@stackific.com` + work key
- In your personal repos: commits use `you@personal.com` + personal key
- Both show **Verified** on GitHub because both keys are registered as signing keys

### Add both keys to allowed_signers (optional)

```bash
echo "you@personal.com $(cat ~/.ssh/id_ed25519_personal.pub)" >> ~/.ssh/allowed_signers
echo "you@stackific.com $(cat ~/.ssh/id_ed25519_work.pub)" >> ~/.ssh/allowed_signers
```

### Verify the right key is active

```bash
# In the specd repo
git config user.email       # should print: you@stackific.com
git config user.signingkey  # should print: ~/.ssh/id_ed25519_work.pub

# In a personal repo
git config user.email       # should print: you@personal.com
git config user.signingkey  # should print: ~/.ssh/id_ed25519_personal.pub
```

---

# Part 3: Troubleshooting

## 3.1 "GitHub shows Unverified on my commits"

Most common causes:

**Cause A: The key is registered as an authentication key only, not a signing key.** Go to https://github.com/settings/keys and confirm there is an entry with **Signing Key** type containing your public key. If not, add one.

**Cause B: The commit email does not match any verified email on your GitHub account.** Go to https://github.com/settings/emails and add/verify the email shown in your commit author line.

**Cause C: You signed with a different key than the one registered.** Check with:

```bash
git config user.signingkey
cat $(git config user.signingkey)
```

Compare that public key text against what is registered at https://github.com/settings/keys. They must match exactly.

**Cause D: Git is too old.** SSH signing requires Git 2.34+. Upgrade.

## 3.2 "error: gpg failed to sign the data" when committing

Despite the error message, this can happen with SSH signing too; the error is generic. Usually means:

- `ssh-keygen` is not on your `PATH`. Fix: `which ssh-keygen`. Install/update OpenSSH if missing.
- `user.signingkey` points to a non-existent file. Fix: check the path is correct and readable.
- You set `gpg.format ssh` but Git is still trying GPG. Fix: `git config --list | grep -i sign` to audit; remove conflicting global configs.

## 3.3 "I use a YubiKey / hardware security key"

SSH signing with a FIDO-backed key (`ed25519-sk`) works the same way. Generate with:

```bash
ssh-keygen -t ed25519-sk -C "you@stackific.com"
```

Register the resulting `.pub` as a signing key on GitHub. Git will prompt for hardware presence (tap the key) at commit time. The key is bound to the device, so exfiltration is impossible.

## 3.4 "My old commits on this PR are unsigned"

Same approach as DCO: rewrite the PR branch. From the PR branch:

```bash
git rebase --exec 'git commit --amend --no-edit -S' $(git merge-base HEAD origin/main)
git push --force-with-lease
```

The `-S` flag forces re-signing of each commit. Commits keep their original author and message; only the signature is added.

If you need to fix signing **and** DCO signoff at the same time:

```bash
git rebase --signoff --exec 'git commit --amend --no-edit -S' \
  $(git merge-base HEAD origin/main)
git push --force-with-lease
```

Only do this on commits you authored. If the PR contains commits by others, do not resign them as you; ask the original author to sign.

## 3.5 "I commit from the GitHub web editor"

The web editor signs commits automatically using GitHub's internal key and marks them as **Verified**. This works for branch protection.

However, web-editor commits do **not** include a `Signed-off-by` line, so they will fail DCO. For `specd`, avoid the web editor; use the CLI.

## 3.6 "I use GitHub Desktop / VS Code / JetBrains"

All of them respect `git config commit.gpgsign true` and the SSH signing config. Once Part 2 is complete, GUI commits sign automatically.

Exception: some older GUI versions have buggy SSH-signing support. If a GUI commit shows as unverified on GitHub, update the tool or commit from the CLI.

## 3.7 "My passphrase is prompting on every commit"

Use the OS keychain:

**macOS:**

```bash
ssh-add --apple-use-keychain ~/.ssh/id_ed25519
```

And in `~/.ssh/config`:

```
Host *
  UseKeychain yes
  AddKeysToAgent yes
  IdentityFile ~/.ssh/id_ed25519
```

**Linux (GNOME):** `gnome-keyring` usually handles this. Otherwise run an agent: `eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519`.

**Windows:** use the built-in OpenSSH agent service (`Get-Service ssh-agent`). Start it and `ssh-add`.

## 3.8 "I lost my laptop / my key is compromised"

1. Go to https://github.com/settings/keys immediately and **Delete** the compromised signing key. This invalidates all verifications for commits signed with that key going forward; historical green badges will turn to **Unverified**.
2. Generate a new key on your new machine (Part 2.1).
3. Register it as a signing key (Part 2.2).
4. Notify `security@stackific.com` so they can audit recent commits attributed to you.

Branch protection is not retroactive, so merged commits stay merged. If you believe a malicious commit was pushed during the compromise window, that is a security incident; escalate.

---

# Part 4: Quick Reference

## Developer cheat sheet

| Situation | Command |
|---|---|
| One-time SSH signing setup | `git config gpg.format ssh && git config user.signingkey ~/.ssh/id_ed25519.pub && git config commit.gpgsign true && git config tag.gpgsign true` |
| Normal commit | `git commit -m "message"` (signing is automatic) |
| Force sign on amend | `git commit --amend -S --no-edit` |
| Re-sign all PR commits | `git rebase --exec 'git commit --amend --no-edit -S' $(git merge-base HEAD origin/main) && git push --force-with-lease` |
| Re-sign + signoff all PR commits | `git rebase --signoff --exec 'git commit --amend --no-edit -S' $(git merge-base HEAD origin/main) && git push --force-with-lease` |
| Inspect signature on last commit | `git log -1 --show-signature` |

## Sysadmin cheat sheet

| Task | Where |
|---|---|
| Enable signing requirement | Repo → Settings → Branches → edit `main` rule → check **Require signed commits** |
| Combined with DCO | Same rule: required status check `DCO` **and** **Require signed commits** |

## Anatomy of a fully compliant `specd` commit

```
Fix parser crash on empty input

Guard against nil input and return a typed error.

Signed-off-by: Jane Doe <jane@stackific.com>
```

- Author email matches a verified email on Jane's GitHub account
- `Signed-off-by` email matches the author email (DCO)
- Commit carries a valid SSH signature matching a signing key registered to Jane's GitHub account (signing)
- Shows green **Verified** badge on GitHub

---

# Part 5: FAQ

**Q: Is SSH signing as secure as GPG signing?**
Functionally equivalent for our threat model. Both produce a cryptographic signature over the commit contents using a key the author controls.

**Q: Do I need a separate key for signing vs authentication?**
No, you can reuse your existing SSH key. Register it twice on GitHub (once as Authentication Key, once as Signing Key). Some security-conscious developers prefer separate keys so that revoking one does not disrupt the other; your call.

**Q: Does signing protect against the code itself being malicious?**
No. Signing proves the commit was made by the key holder, not that the content is safe. Code review still matters.

**Q: What about commits signed by GitHub itself (web edits, merges, suggestions)?**
GitHub signs them with its own key and they show as **Verified** by `github.com`. These satisfy branch protection but will fail DCO, so avoid them for `specd`.

**Q: Can I sign commits from multiple machines?**
Yes. Generate a separate signing key on each machine and register each as a Signing Key on your GitHub account. Never copy private keys between machines.

**Q: How do I handle signing in CI (bots, automation)?**
Bots need their own signing key stored as a CI secret, or they commit through GitHub's API which signs automatically. Raise with the CI maintainer.

**Q: What happens to my old unsigned commits in `main`?**
They stay unsigned. Branch protection applies only to new commits. Rewriting `main` history to back-sign old commits is not worth the disruption and is not standard practice.

**Q: Can I opt out of signing for a single commit?**
`git commit --no-gpg-sign` skips signing. The commit will then be blocked by branch protection on `main`. Useful only for local experiments.

---

# Appendix A: GPG Signing (alternative to SSH)

Use this path only if you already have an established GPG setup or have a specific reason to prefer GPG. Otherwise, use SSH (Part 2).

## A.1 Install GPG

- macOS: `brew install gnupg`
- Linux: usually preinstalled; else `apt install gnupg` or equivalent
- Windows: [Gpg4win](https://gpg4win.org/)

## A.2 Generate a key

```bash
gpg --full-generate-key
```

- Type: **ECC (sign and encrypt)** or RSA 4096
- Curve: **Curve 25519**
- Expiration: 2 years (recommended; you can extend later)
- Name: your full name
- Email: `you@stackific.com` (must match a verified GitHub email)
- Passphrase: strong, cached in keychain

## A.3 Export the public key and register on GitHub

```bash
# List keys to find the key ID
gpg --list-secret-keys --keyid-format=long
# Output contains a line like: sec   ed25519/ABCDEF1234567890 ...

# Export the public key
gpg --armor --export ABCDEF1234567890 | pbcopy   # macOS
gpg --armor --export ABCDEF1234567890            # Linux/Windows - copy output manually
```

Go to https://github.com/settings/keys, click **New GPG key**, paste, save.

## A.4 Configure Git

```bash
git config gpg.format openpgp         # default; explicit for clarity
git config user.signingkey ABCDEF1234567890
git config commit.gpgsign true
git config tag.gpgsign true
```

## A.5 Verify

Same as Part 2.5. `git log --show-signature` should show a good GPG signature; GitHub should display **Verified**.

## A.6 Passphrase caching

- macOS: install `pinentry-mac` (`brew install pinentry-mac`), configure `~/.gnupg/gpg-agent.conf` with `pinentry-program /opt/homebrew/bin/pinentry-mac`.
- Linux: `gpg-agent` runs by default; configure `default-cache-ttl` in `~/.gnupg/gpg-agent.conf`.
- Windows: Gpg4win's Kleopatra handles this.

## A.7 Key rotation

GPG keys have expiration dates. When yours approaches expiry:

```bash
gpg --edit-key ABCDEF1234567890
> expire
> (set new expiration)
> save

# Re-export and replace the key on GitHub
gpg --armor --export ABCDEF1234567890
```

Never delete the old key locally; you will need it to verify your own historical commits.

---

*Document owner: Stackific Engineering. Last reviewed: 2026-04-21. For questions, contact info@stackific.com. For compromised keys, contact security@stackific.com.*