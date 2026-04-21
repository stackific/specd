# specd uninstaller for Windows — Stackific Inc. All rights reserved.
# https://stackific.com/specd
# Usage: irm https://stackific.com/specd/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"

$Product = "specd"
$InstallDir = "$env:USERPROFILE\.specd"
$BinDir = "$InstallDir\bin"

function Main {
    if (-not (Test-Path $BinDir)) {
        Write-Host "$Product is not installed ($BinDir not found)."
        return
    }

    Write-Host "Removing $BinDir..."
    Remove-Item -Recurse -Force $BinDir
    Write-Host "Removed $BinDir"

    Write-Host ""
    Write-Host "Note: $InstallDir\ (config, cache, skills) was kept."
    Write-Host "To remove everything: Remove-Item -Recurse -Force $InstallDir"

    Clean-Path

    Write-Host ""
    Write-Host "$Product has been uninstalled." -ForegroundColor Green
}

function Clean-Path {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = $currentPath -split ";" | Where-Object { $_ -ne $BinDir -and $_ -ne "" }

    if ($parts.Count -lt ($currentPath -split ";").Count) {
        $newPath = $parts -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "Removed $BinDir from PATH."
        Write-Host "Open a new terminal for changes to take effect."
    }
}

Main
