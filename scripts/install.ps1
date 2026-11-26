$ErrorActionPreference = "Stop"

$Repo = "eraygdev/reta-toolkit"
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq "Arm64") { "arm64" } else { "amd64" }
$Asset = "reta-windows-$Arch.exe"
$Url = "https://github.com/$Repo/releases/latest/download/$Asset"

$InstallDir = Join-Path $env:LOCALAPPDATA "reta\bin"
$BinPath = Join-Path $InstallDir "reta.exe"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Write-Host "Downloading $Asset..."
Invoke-WebRequest -Uri $Url -OutFile $BinPath -UseBasicParsing

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not ($UserPath -split ";" | Where-Object { $_ -eq $InstallDir })) {
    $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "Added $InstallDir to User PATH"
}

& $BinPath doctor --fix | Out-Null

$ProfileDir = Split-Path -Parent $PROFILE
if ($ProfileDir -and -not (Test-Path $ProfileDir)) {
    New-Item -ItemType Directory -Force -Path $ProfileDir | Out-Null
}
if (-not (Test-Path $PROFILE)) {
    New-Item -ItemType File -Force -Path $PROFILE | Out-Null
}
$ProfileContent = Get-Content $PROFILE -Raw -ErrorAction SilentlyContinue
if (-not ($ProfileContent -match "RETA\*")) {
    Add-Content -Path $PROFILE -Value "`n# ===RETA*===`nif (Get-Command reta -ErrorAction SilentlyContinue) { Invoke-Expression (& reta shell-init powershell | Out-String) }`n# ==========="
    Write-Host "Added rcd integration to $PROFILE"
}

$CurrentPolicy = Get-ExecutionPolicy -ErrorAction SilentlyContinue
if ($CurrentPolicy -eq "Restricted") {
    Write-Host "Warning: PowerShell ExecutionPolicy is 'Restricted'. To allow rcd to load in new terminals, run:" -ForegroundColor Yellow
    Write-Host "         Set-ExecutionPolicy -Scope CurrentUser RemoteSigned" -ForegroundColor Cyan
}

Write-Host "Installed reta to $BinPath"
Write-Host "Run 'reta doctor' in a new PowerShell window to verify."