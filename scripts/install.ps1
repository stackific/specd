# specd installer for Windows
# Usage: irm https://stackific.com/specd/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "stackific/specd"
$Binary = "specd"
$InstallDir = "$env:USERPROFILE\.specd\bin"

function Main {
    $platform = Detect-Platform
    $version = Get-LatestVersion
    Download-Binary -Version $version -Platform $platform
    Install-Binary
    Setup-Path
    Write-Host ""
    Write-Host "specd $version installed successfully!" -ForegroundColor Green
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

    $filename = "$Binary-windows-$Platform.exe"
    $url = "https://github.com/$Repo/releases/download/$Version/$filename"

    Write-Host "Downloading $url..."

    $tmpFile = Join-Path $env:TEMP "$Binary.exe"
    Invoke-WebRequest -Uri $url -OutFile $tmpFile -UseBasicParsing

    $script:TmpFile = $tmpFile
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
