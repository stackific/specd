# npm & pip Distribution Setup

Instructions for publishing specd as an npm package and a Python/pipx package. These are wrapper packages that download and install the correct Go binary for the user's platform.

## npm: `@stackific/specd-cli`

### How it works

The npm package contains no Go code. It has a `postinstall` script that downloads the correct binary from GitHub Releases and places it in the package's `bin/` directory. npm handles PATH via its own bin linking.

### Directory structure

```
npm/
  package.json
  install.js       # postinstall script that downloads the binary
  bin/
    specd           # placeholder, replaced by postinstall
```

### package.json

```json
{
  "name": "@stackific/specd-cli",
  "version": "0.1.0",
  "description": "specd - a specification-driven development CLI tool",
  "bin": {
    "specd": "./bin/specd"
  },
  "scripts": {
    "postinstall": "node install.js"
  },
  "os": ["darwin", "linux", "win32"],
  "cpu": ["x64", "arm64"],
  "keywords": ["specd", "cli", "specification-driven"],
  "license": "UNLICENSED",
  "repository": {
    "type": "git",
    "url": "https://github.com/stackific/specd.git"
  }
}
```

### install.js

```js
const os = require("os");
const fs = require("fs");
const path = require("path");
const https = require("https");
const { execSync } = require("child_process");

const REPO = "stackific/specd";
const BINARY = "specd";
const VERSION = `v${require("./package.json").version}`;

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

async function main() {
  const platform = PLATFORM_MAP[os.platform()];
  const arch = ARCH_MAP[os.arch()];

  if (!platform || !arch) {
    console.error(`Unsupported platform: ${os.platform()}/${os.arch()}`);
    process.exit(1);
  }

  const ext = platform === "windows" ? ".exe" : "";
  const filename = `${BINARY}-${platform}-${arch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/${VERSION}/${filename}`;

  const binDir = path.join(__dirname, "bin");
  if (!fs.existsSync(binDir)) fs.mkdirSync(binDir, { recursive: true });

  const dest = path.join(binDir, `${BINARY}${ext}`);

  console.log(`Downloading ${BINARY} ${VERSION} for ${platform}/${arch}...`);

  // Use curl/wget for simplicity with redirects
  try {
    execSync(`curl -sSL -o "${dest}" "${url}"`, { stdio: "inherit" });
  } catch {
    execSync(`wget -qO "${dest}" "${url}"`, { stdio: "inherit" });
  }

  if (platform !== "windows") {
    fs.chmodSync(dest, 0o755);
  }

  console.log(`Installed ${BINARY} to ${dest}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
```

### Publishing to npm

1. Create an npm organization: `@stackific` at https://www.npmjs.com/org/create
2. Log in: `npm login`
3. From the `npm/` directory: `npm publish --access public`

### Version sync

The npm package version must match the GitHub Release version. When releasing:

1. Update `npm/package.json` version to match the release version
2. Run `npm publish --access public` from the `npm/` directory

This can be automated in the GitHub Actions release workflow by adding a job:

```yaml
  publish-npm:
    needs: release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          registry-url: https://registry.npmjs.org
      - name: Update version
        working-directory: npm
        run: npm version ${{ inputs.version }} --no-git-tag-version
      - name: Publish
        working-directory: npm
        run: npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

**Required secret:** `NPM_TOKEN` — generate at https://www.npmjs.com/settings/tokens (Automation type).

---

## pip/pipx: `specd`

### How it works

Same approach as npm: a Python package with a post-install hook that downloads the correct binary. The package itself is pure Python — no Go compilation needed.

### Directory structure

```
python/
  pyproject.toml
  specd_cli/
    __init__.py
    __main__.py       # entry point: runs the Go binary
    install.py        # download logic, called during build
```

### pyproject.toml

```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "specd"
version = "0.1.0"
description = "specd - a specification-driven development CLI tool"
readme = "README.md"
license = "LicenseRef-Proprietary"
requires-python = ">=3.8"
classifiers = [
    "Programming Language :: Python :: 3",
    "Environment :: Console",
]

[project.scripts]
specd = "specd_cli.__main__:main"

[project.urls]
Homepage = "https://github.com/stackific/specd"
```

### specd_cli/\_\_main\_\_.py

```python
"""Entry point that delegates to the Go binary."""
import os
import sys
import subprocess


def main():
    binary = os.path.join(os.path.dirname(__file__), "bin", "specd")
    if sys.platform == "win32":
        binary += ".exe"

    if not os.path.exists(binary):
        print("Error: specd binary not found. Try reinstalling: pipx install specd", file=sys.stderr)
        sys.exit(1)

    result = subprocess.run([binary] + sys.argv[1:])
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
```

### specd_cli/\_\_init\_\_.py

```python
"""specd CLI wrapper package."""
```

### Build hook for downloading the binary

Use a custom hatch build hook. Create `hatch_build.py` at the root of the `python/` directory:

```python
"""Custom build hook to download the Go binary during wheel build."""
import os
import platform
import subprocess
import stat
from hatchling.builders.hooks.plugin.interface import BuildHookInterface


REPO = "stackific/specd"
BINARY = "specd"


class DownloadBinaryHook(BuildHookInterface):
    def initialize(self, version, build_data):
        system = platform.system().lower()
        machine = platform.machine().lower()

        os_map = {"linux": "linux", "darwin": "darwin", "windows": "windows"}
        arch_map = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}

        os_name = os_map.get(system)
        arch = arch_map.get(machine)
        if not os_name or not arch:
            raise RuntimeError(f"Unsupported platform: {system}/{machine}")

        ext = ".exe" if os_name == "windows" else ""
        filename = f"{BINARY}-{os_name}-{arch}{ext}"
        url = f"https://github.com/{REPO}/releases/download/v{version}/{filename}"

        bin_dir = os.path.join(self.root, "specd_cli", "bin")
        os.makedirs(bin_dir, exist_ok=True)
        dest = os.path.join(bin_dir, f"{BINARY}{ext}")

        subprocess.run(["curl", "-sSL", "-o", dest, url], check=True)
        if os_name != "windows":
            os.chmod(dest, os.stat(dest).st_mode | stat.S_IEXEC)

        build_data["shared_data"]["specd_cli/bin"] = "specd_cli/bin"
```

Update `pyproject.toml` to register the hook:

```toml
[tool.hatch.build.hooks.custom]
path = "hatch_build.py"
```

### Publishing to PyPI

1. Create a PyPI account at https://pypi.org/account/register/
2. Create an API token at https://pypi.org/manage/account/token/
3. Build and publish:

```sh
cd python/
pip install build twine
python -m build
twine upload dist/*
```

### Version sync

Same as npm — the version in `pyproject.toml` must match the GitHub Release version.

Automate in the release workflow:

```yaml
  publish-pypi:
    needs: release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - name: Update version
        working-directory: python
        run: sed -i "s/^version = .*/version = \"${{ inputs.version }}\"/" pyproject.toml
      - name: Build and publish
        working-directory: python
        run: |
          pip install build twine
          python -m build
          twine upload dist/*
        env:
          TWINE_USERNAME: __token__
          TWINE_PASSWORD: ${{ secrets.PYPI_TOKEN }}
```

**Required secret:** `PYPI_TOKEN` — generate at https://pypi.org/manage/account/token/.

---

## Summary of Required Secrets

| Secret       | Where to create                          | Used for       |
| ------------ | ---------------------------------------- | -------------- |
| `NPM_TOKEN`  | https://www.npmjs.com/settings/tokens    | npm publish    |
| `PYPI_TOKEN`  | https://pypi.org/manage/account/token/   | pip publish    |

Both are added at: GitHub repo > Settings > Secrets and variables > Actions > New repository secret.

## Checklist

- [ ] Create `@stackific` npm org
- [ ] Create `npm/` directory with `package.json` and `install.js`
- [ ] Register PyPI account
- [ ] Create `python/` directory with `pyproject.toml` and wrapper code
- [ ] Add `NPM_TOKEN` and `PYPI_TOKEN` to GitHub repo secrets
- [ ] Add publish jobs to the release workflow
- [ ] Test: `npm install -g @stackific/specd-cli` installs and runs `specd`
- [ ] Test: `pipx install specd` installs and runs `specd`
