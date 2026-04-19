# specd uninstaller for Windows
# Usage: irm https://stackific.com/specd/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"

$InstallDir = "$env:USERPROFILE\.specd"

function Main {
    if (-not (Test-Path $InstallDir)) {
        Write-Host "specd is not installed ($InstallDir not found)."
        return
    }

    Write-Host "Removing $InstallDir..."
    Remove-Item -Recurse -Force $InstallDir
    Write-Host "Removed $InstallDir"

    Clean-Path

    Write-Host ""
    Write-Host "specd has been uninstalled." -ForegroundColor Green
}

function Clean-Path {
    $binDir = "$InstallDir\bin"
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = $currentPath -split ";" | Where-Object { $_ -ne $binDir -and $_ -ne "" }

    if ($parts.Count -lt ($currentPath -split ";").Count) {
        $newPath = $parts -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "Removed $binDir from PATH."
        Write-Host "Open a new terminal for changes to take effect."
    }
}

Main
