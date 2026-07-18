# Copyright 2026 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

[CmdletBinding()]
param(
    [string]$Prefix = $(if ($env:GOPLUS_PREFIX) { $env:GOPLUS_PREFIX } else { Join-Path $env:LOCALAPPDATA "GoPlus" }),
    [switch]$NoPathUpdate
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT") {
    throw "install-goplus.ps1 must run on Windows"
}

function Invoke-Native {
    param(
        [Parameter(Mandatory)][string]$File,
        [Parameter(Mandatory)][string[]]$Arguments
    )
    & $File @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$File exited with status $LASTEXITCODE"
    }
}

if (-not (Get-Command git.exe -ErrorAction SilentlyContinue)) {
    throw "git.exe is required"
}

$RepoRoot = $PSScriptRoot
$Prefix = [IO.Path]::GetFullPath($Prefix)
$PrefixParent = Split-Path -Parent $Prefix
$HomePath = [IO.Path]::GetFullPath($HOME).TrimEnd('\')
$DriveRoot = [IO.Path]::GetPathRoot($Prefix).TrimEnd('\')
if (-not $PrefixParent -or $Prefix.TrimEnd('\') -eq $HomePath -or $Prefix.TrimEnd('\') -eq $DriveRoot) {
    throw "refusing unsafe install prefix: $Prefix"
}

$ToolsRepo = if ($env:GOPLUS_TOOLS_REPO) { $env:GOPLUS_TOOLS_REPO } else { "https://go.googlesource.com/tools" }
$ToolsRef = if ($env:GOPLUS_TOOLS_REF) { $env:GOPLUS_TOOLS_REF } else { "635ae9663724" }
$GoplsRef = if ($env:GOPLUS_GOPLS_REF) { $env:GOPLUS_GOPLS_REF } else { "gopls/v0.22.0" }

Write-Host "Building Go+ from $RepoRoot"
Push-Location (Join-Path $RepoRoot "src")
try {
    Invoke-Native -File "cmd.exe" -Arguments @("/d", "/c", "make.bat")
} finally {
    Pop-Location
}

$BuiltGo = Join-Path $RepoRoot "bin\go.exe"
$BuiltGofmt = Join-Path $RepoRoot "bin\gofmt.exe"
$VersionCache = Join-Path $RepoRoot "VERSION.cache"
foreach ($Required in @($BuiltGo, $BuiltGofmt, $VersionCache)) {
    if (-not (Test-Path -LiteralPath $Required)) {
        throw "build did not produce $Required"
    }
}

New-Item -ItemType Directory -Force -Path $PrefixParent | Out-Null
$Work = Join-Path $PrefixParent (".goplus-install-" + [guid]::NewGuid().ToString("N"))
$Stage = Join-Path $Work "root"
$StageGo = Join-Path $Stage "go"
$StageBin = Join-Path $Stage "bin"
$StageLibexec = Join-Path $Stage "libexec"
New-Item -ItemType Directory -Force -Path $StageGo, $StageBin, $StageLibexec | Out-Null

$OriginalGOROOT = $env:GOROOT
$OriginalGOTOOLCHAIN = $env:GOTOOLCHAIN
$OriginalGOWORK = $env:GOWORK

try {
    foreach ($Dir in @("api", "bin", "doc", "lib", "misc", "src")) {
        Copy-Item -Recurse -Force -LiteralPath (Join-Path $RepoRoot $Dir) -Destination $StageGo
    }
    $StagePkg = Join-Path $StageGo "pkg"
    New-Item -ItemType Directory -Force -Path $StagePkg | Out-Null
    Copy-Item -Recurse -Force -LiteralPath (Join-Path $RepoRoot "pkg\include") -Destination $StagePkg
    Copy-Item -Recurse -Force -LiteralPath (Join-Path $RepoRoot "pkg\tool") -Destination $StagePkg
    Copy-Item -Force -LiteralPath (Join-Path $RepoRoot "go.env") -Destination (Join-Path $StageGo "go.env")
    Copy-Item -Force -LiteralPath $VersionCache -Destination (Join-Path $StageGo "VERSION")

    $ToolsDir = Join-Path $Work "tools"
    Write-Host "Fetching patched x/tools ($ToolsRef)"
    Invoke-Native -File "git.exe" -Arguments @("clone", "--quiet", "--filter=blob:none", "--no-checkout", $ToolsRepo, $ToolsDir)
    Invoke-Native -File "git.exe" -Arguments @("-C", $ToolsDir, "checkout", "--quiet", $ToolsRef)
    Invoke-Native -File "git.exe" -Arguments @("-C", $ToolsDir, "apply", (Join-Path $RepoRoot "misc\enum\x-tools.patch"))

    $GoplsRepo = Join-Path $Work "gopls-repo"
    Write-Host "Fetching patched gopls ($GoplsRef)"
    Invoke-Native -File "git.exe" -Arguments @("clone", "--quiet", "--filter=blob:none", "--no-checkout", $ToolsRepo, $GoplsRepo)
    Invoke-Native -File "git.exe" -Arguments @("-C", $GoplsRepo, "checkout", "--quiet", $GoplsRef)
    $GoplsDir = Join-Path $GoplsRepo "gopls"
    Invoke-Native -File "git.exe" -Arguments @("-C", $GoplsRepo, "apply", "--directory=gopls", (Join-Path $RepoRoot "misc\enum\gopls.patch"))

    $PrivateGo = Join-Path $StageGo "bin\go.exe"
    $env:GOROOT = $StageGo
    $env:GOTOOLCHAIN = "local"
    $env:GOWORK = "off"

    Push-Location $ToolsDir
    try {
        Invoke-Native -File $PrivateGo -Arguments @("build", "-trimpath", "-o", (Join-Path $StageLibexec "goimports.exe"), "./cmd/goimports")
    } finally {
        Pop-Location
    }

    Push-Location $GoplsDir
    try {
        Invoke-Native -File $PrivateGo -Arguments @("mod", "edit", "-replace=golang.org/x/tools=$ToolsDir")
        Invoke-Native -File $PrivateGo -Arguments @("build", "-trimpath", "-o", (Join-Path $StageLibexec "gopls.exe"), ".")
    } finally {
        Pop-Location
    }

    $Launcher = Join-Path $StageBin "goplus-launcher.exe"
    Push-Location (Join-Path $RepoRoot "misc")
    try {
        Invoke-Native -File $PrivateGo -Arguments @("build", "-trimpath", "-o", $Launcher, "./enum/goplus-launcher")
    } finally {
        Pop-Location
    }
    foreach ($Name in @("go+", "gofmt+", "gopls+", "goimports+")) {
        Copy-Item -Force -LiteralPath $Launcher -Destination (Join-Path $StageBin "$Name.exe")
    }
    Remove-Item -Force -LiteralPath $Launcher

    Invoke-Native -File (Join-Path $StageBin "go+.exe") -Arguments @("version")
    Invoke-Native -File (Join-Path $StageBin "gopls+.exe") -Arguments @("version")

    $Backup = $null
    try {
        if (Test-Path -LiteralPath $Prefix) {
            $Backup = "$Prefix.backup.$PID"
            if (Test-Path -LiteralPath $Backup) {
                throw "backup path already exists: $Backup"
            }
            Move-Item -LiteralPath $Prefix -Destination $Backup
        }
        Move-Item -LiteralPath $Stage -Destination $Prefix
        if ($Backup) {
            Remove-Item -Recurse -Force -LiteralPath $Backup
        }
    } catch {
        if ($Backup -and -not (Test-Path -LiteralPath $Prefix) -and (Test-Path -LiteralPath $Backup)) {
            Move-Item -LiteralPath $Backup -Destination $Prefix
        }
        throw
    }

    $BinDir = Join-Path $Prefix "bin"
    if (-not $NoPathUpdate) {
        $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
        $Entries = @($UserPath -split ';' | Where-Object { $_ })
        if (-not ($Entries | Where-Object { $_.TrimEnd('\') -ieq $BinDir.TrimEnd('\') })) {
            $NewPath = if ($UserPath) { "$BinDir;$UserPath" } else { $BinDir }
            [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        }
        $env:Path = "$BinDir;$env:Path"
    }

    Write-Host ""
    Write-Host "Go+ installed in $Prefix"
    Write-Host "Commands: go+, gofmt+, gopls+, goimports+"
    if ($NoPathUpdate) {
        Write-Host "Add $BinDir to your user PATH."
    } else {
        Write-Host "Open a new terminal to use the updated user PATH."
    }
} finally {
    if ($null -eq $OriginalGOROOT) {
        Remove-Item Env:GOROOT -ErrorAction SilentlyContinue
    } else {
        $env:GOROOT = $OriginalGOROOT
    }
    if ($null -eq $OriginalGOTOOLCHAIN) {
        Remove-Item Env:GOTOOLCHAIN -ErrorAction SilentlyContinue
    } else {
        $env:GOTOOLCHAIN = $OriginalGOTOOLCHAIN
    }
    if ($null -eq $OriginalGOWORK) {
        Remove-Item Env:GOWORK -ErrorAction SilentlyContinue
    } else {
        $env:GOWORK = $OriginalGOWORK
    }
    if (Test-Path -LiteralPath $Work) {
        Remove-Item -Recurse -Force -LiteralPath $Work
    }
}
