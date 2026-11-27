$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$BinPath = if ($env:RETA_BIN) { $env:RETA_BIN } else { Join-Path $RepoRoot "reta.exe" }
if (-not (Test-Path $BinPath)) {
    throw "reta binary not found at $BinPath"
}
$BinPath = (Resolve-Path $BinPath).Path

$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("reta-win-runtime-" + [guid]::NewGuid().ToString("N"))
$OrigHome = $env:HOME
$OrigUserProfile = $env:USERPROFILE
$OrigLocalAppData = $env:LOCALAPPDATA
$OrigNoColor = $env:NO_COLOR

try {
    New-Item -ItemType Directory -Force -Path $TempRoot | Out-Null
    $FakeHome = Join-Path $TempRoot "home"
    $WorkDir = Join-Path $TempRoot "proj"
    $LocalAppData = Join-Path $FakeHome "AppData\Local"
    New-Item -ItemType Directory -Force -Path $FakeHome, $WorkDir, $LocalAppData | Out-Null

    $env:HOME = $FakeHome
    $env:USERPROFILE = $FakeHome
    $env:LOCALAPPDATA = $LocalAppData
    $env:NO_COLOR = "1"
    $env:RETA_HINTS = "off"

    # 1. Version check
    $ver = (& $BinPath --version).Trim()
    $expectedVer = (Get-Content (Join-Path $RepoRoot "VERSION") -Raw).Trim()
    if ($ver -ne $expectedVer) {
        throw "Version mismatch: got '$ver', expected '$expectedVer'"
    }

    # 2. PowerShell shell-init and completion AST validation
    foreach ($sub in @("shell-init", "completion")) {
        $script = (& $BinPath $sub powershell | Out-String)
        $errors = $null
        $null = [System.Management.Automation.Language.Parser]::ParseInput($script, [ref]$null, [ref]$errors)
        if ($errors.Count -gt 0) {
            throw "PowerShell syntax error in '$sub powershell': $($errors[0].Message)"
        }
    }

    # 3. Project init, init-env, env check, check-secrets, export
    Push-Location $WorkDir
    try {
        & $BinPath init . | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "reta init failed" }
        foreach ($f in @(".env.example", ".gitignore", "Makefile", "README.md", ".retasign", "AGENTS.md")) {
            if (-not (Test-Path (Join-Path $WorkDir $f))) { throw "Missing scaffolded file: $f" }
        }

        [System.IO.File]::WriteAllText((Join-Path $WorkDir ".env.example"), "APP_KEY=GENERATE_SECRET`nSTATIC=ok`n", [System.Text.UTF8Encoding]::new($false))
        & $BinPath init-env | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "reta init-env failed" }
        & $BinPath env check | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "reta env check failed" }
        & $BinPath check-secrets | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "reta check-secrets failed" }
        & $BinPath export | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "reta export failed" }
        if (-not (Test-Path (Join-Path $WorkDir "exports\project_export.md"))) {
            throw "Missing exports\project_export.md"
        }
    } finally {
        Pop-Location
    }

    # 4. Live rcd jump in PowerShell
    $env:Path = (Split-Path -Parent $BinPath) + ";" + $env:Path
    $initCode = (& $BinPath shell-init powershell | Out-String)
    Invoke-Expression $initCode
    Push-Location $FakeHome
    try {
        rcd proj | Out-Null
        $landed = (Get-Location).Path
        if (-not (Test-Path (Join-Path $landed ".retasign")) -or ((Split-Path -Leaf $landed) -ne "proj")) {
            throw "rcd landed at '$landed', expected '$WorkDir'"
        }
    } finally {
        Pop-Location
    }

    # 5. Live TCP socket with reta port list
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    try {
        $port = ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port.ToString()
        $portOut = (& $BinPath port list $port | Out-String)
        if ($portOut -notmatch "\b$port\b") {
            throw "reta port list $port did not report active listener:`n$portOut"
        }
    } finally {
        $listener.Stop()
    }

    # 6. doctor --fix creates base.mk
    & $BinPath doctor --fix | Out-Null
    $baseMK = Join-Path $FakeHome ".reta\lib\base.mk"
    if (-not (Test-Path $baseMK)) {
        throw "doctor --fix did not create $baseMK"
    }

    Write-Host "Windows runtime tests passed." -ForegroundColor Green
} finally {
    $env:HOME = $OrigHome
    $env:USERPROFILE = $OrigUserProfile
    $env:LOCALAPPDATA = $OrigLocalAppData
    $env:NO_COLOR = $OrigNoColor
    Remove-Item -Recurse -Force $TempRoot -ErrorAction SilentlyContinue
}