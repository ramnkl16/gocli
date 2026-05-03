# Install latest gocli from GitHub Releases into %LOCALAPPDATA%\Programs\gocli and add User PATH.
# Usage (PowerShell):
#   iex "& { $(irm https://raw.githubusercontent.com/YOUR_ORG/gocli/main/scripts/install.ps1) }"
# Override repo:
#   $env:GOCLI_GITHUB_REPO = 'myorg/ezcli'; iex "& { $(irm ...) }"

$ErrorActionPreference = 'Stop'

$Repo = $env:GOCLI_GITHUB_REPO
if ([string]::IsNullOrWhiteSpace($Repo)) {
    $Repo = 'yourorg/gocli'
}

$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$Asset = "gocli_Windows_$Arch.zip"
$Url = "https://github.com/$Repo/releases/latest/download/$Asset"
$InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\gocli'
$ExePath = Join-Path $InstallDir 'gocli.exe'

Write-Host "Downloading $Url"
$tmpZip = Join-Path ([System.IO.Path]::GetTempPath()) ("gocli-" + [Guid]::NewGuid().ToString() + '.zip')
$tmpExtract = Join-Path ([System.IO.Path]::GetTempPath()) ('gocli-extract-' + [Guid]::NewGuid().ToString())

try {
    Invoke-WebRequest -Uri $Url -OutFile $tmpZip -UseBasicParsing
    New-Item -ItemType Directory -Path $tmpExtract -Force | Out-Null
    Expand-Archive -Path $tmpZip -DestinationPath $tmpExtract -Force

    Get-ChildItem -Path $tmpExtract -Filter 'gocli.exe' -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1 | ForEach-Object { $_.FullName } | ForEach-Object {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Copy-Item -Path $_ -Destination $ExePath -Force
    }
}
finally {
    Remove-Item -LiteralPath $tmpZip -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $tmpExtract -Recurse -Force -ErrorAction SilentlyContinue
}

if (-not (Test-Path -LiteralPath $ExePath)) {
    Write-Error "gocli.exe not found in downloaded archive. Check that $Repo has a release with asset $Asset"
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*${InstallDir}*") {
    $newValue = if ([string]::IsNullOrWhiteSpace($userPath)) { $InstallDir } else { "$InstallDir;$userPath" }
    [Environment]::SetEnvironmentVariable('Path', $newValue, 'User')
    Write-Host "Added $InstallDir to your user PATH. Open a new terminal for it to take effect."
}
else {
    Write-Host "$InstallDir is already on your user PATH."
}

Write-Host "Installed: $ExePath"
& $ExePath version
