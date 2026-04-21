# specd installer for Windows — Stackific Inc. All rights reserved.
# https://stackific.com/specd
# Usage: irm https://stackific.com/specd/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Company = "Stackific Inc."
$Product = "specd"
$Homepage = "https://stackific.com/specd"
$Repo = "stackific/specd"
$Binary = "specd"
$InstallDir = "$env:USERPROFILE\.specd\bin"

function Main {
    Write-Host "$Product installer - $Company"
    Write-Host "$Homepage"
    Write-Host ""
    $platform = Detect-Platform
    $version = Get-LatestVersion
    Download-Binary -Version $version -Platform $platform
    Verify-Checksum -Platform $platform
    Install-Binary
    Setup-Path
    Write-Host ""
    Write-Host "$Product $version installed successfully!" -ForegroundColor Green
    Write-Host ""
}

function Detect-Platform {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Get-LatestVersion {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $version = $release.tag_name
    if (-not $version) {
        Write-Error "Could not determine latest version"
        exit 1
    }
    Write-Host "Latest version: $version"
    return $version
}

function Download-Binary {
    param($Version, $Platform)

    $script:Filename = "$Binary-windows-$Platform.exe"
    $binaryUrl = "https://github.com/$Repo/releases/download/$Version/$($script:Filename)"
    $checksumsUrl = "https://github.com/$Repo/releases/download/$Version/checksums.txt"

    Write-Host "Downloading $binaryUrl..."

    $script:TmpFile = Join-Path $env:TEMP $script:Filename
    $script:TmpChecksums = Join-Path $env:TEMP "specd-checksums.txt"

    try {
        Invoke-WebRequest -Uri $binaryUrl -OutFile $script:TmpFile -UseBasicParsing
        Invoke-WebRequest -Uri $checksumsUrl -OutFile $script:TmpChecksums -UseBasicParsing
    } catch {
        Write-Error "Download failed: $_"
        exit 1
    }
}

function Verify-Checksum {
    param($Platform)

    Write-Host "Verifying checksum..."

    $checksumLines = Get-Content $script:TmpChecksums
    $expectedLine = $checksumLines | Where-Object { $_ -match "$($script:Filename)$" }
    if (-not $expectedLine) {
        Write-Error "Binary not found in checksums.txt"
        exit 1
    }
    $expected = ($expectedLine -split "\s+")[0]

    $actual = (Get-FileHash -Path $script:TmpFile -Algorithm SHA256).Hash.ToLower()

    if ($expected -ne $actual) {
        Write-Error "Checksum mismatch`n  Expected: $expected`n  Actual:   $actual"
        exit 1
    }

    Write-Host "Checksum verified."
    Remove-Item $script:TmpChecksums -Force
}

function Install-Binary {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $dest = Join-Path $InstallDir "$Binary.exe"
    Move-Item -Path $script:TmpFile -Destination $dest -Force
    Write-Host "Installed to $dest"
}

function Setup-Path {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")

    if ($currentPath -split ";" | Where-Object { $_ -eq $InstallDir }) {
        Write-Host "PATH already configured."
        return
    }

    Write-Host ""
    $response = Read-Host "Add $InstallDir to your PATH? (Y/n)"
    if ($response -eq "" -or $response -match "^[Yy]") {
        $newPath = "$InstallDir;$currentPath"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$InstallDir;$env:Path"
        Write-Host "PATH updated. Changes apply to new terminal windows." -ForegroundColor Green
        Write-Host "Run 'specd' in a new terminal to get started."
    } else {
        Write-Host ""
        Write-Host "To add manually, run:"
        Write-Host "  `$env:Path = `"$InstallDir;`$env:Path`""
        Write-Host ""
        Write-Host "To make it permanent:"
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', '$InstallDir;' + [Environment]::GetEnvironmentVariable('Path', 'User'), 'User')"
    }
}

Main
