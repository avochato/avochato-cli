# install.ps1: download, verify, and install the avochato CLI on Windows.
# Usage: irm https://raw.githubusercontent.com/avochato/avochato-cli/master/install.ps1 | iex
# INSTALL_DIR picks the destination (default %LOCALAPPDATA%\Programs\avochato, added to your user PATH);
# VERSION pins a release tag. Set them on $env: before piping to iex, e.g. $env:VERSION='v0.1.0'.

$ErrorActionPreference = 'Stop'
# Windows PowerShell 5.1 downloads are many times slower with the progress bar on.
$ProgressPreference = 'SilentlyContinue'

# Windows PowerShell 5.1 may default to TLS 1.0, which GitHub rejects.
try {
  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
} catch { }

$Repo = 'avochato/avochato-cli'
$Binary = 'avochato'
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\avochato' }
$Version = $env:VERSION

function Fail([string]$Message) { throw $Message }

function Get-Arch {
  $arch = $env:PROCESSOR_ARCHITECTURE
  if ($env:PROCESSOR_ARCHITEW6432) { $arch = $env:PROCESSOR_ARCHITEW6432 }
  switch -Regex ($arch) {
    '^(AMD64|x86_64)$' { return 'amd64' }
    '^ARM64$' { return 'arm64' }
    default { Fail "Unsupported architecture: $arch. Download manually from https://github.com/$Repo/releases" }
  }
}

function Get-File([string]$Url, [string]$Destination) {
  # -UseBasicParsing keeps Windows PowerShell 5.1 from loading Internet Explorer's parser; no-op on 6+.
  Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Destination `
    -Headers @{ 'User-Agent' = 'avochato-cli-installer' }
}

function Test-Checksum([string]$ChecksumsPath, [string]$ArchivePath, [string]$ArchiveName) {
  $expected = $null
  foreach ($line in Get-Content $ChecksumsPath) {
    if ($line -match '^(?<hash>[0-9a-fA-F]{64})\s+\*?(?<name>\S+)$' -and $Matches.name -eq $ArchiveName) {
      $expected = $Matches.hash.ToLowerInvariant()
      break
    }
  }
  if (-not $expected) { Fail "No checksum found for $ArchiveName" }

  $actual = (Get-FileHash -Algorithm SHA256 -Path $ArchivePath).Hash.ToLowerInvariant()
  if ($actual -ne $expected) { Fail "Checksum mismatch for $ArchiveName" }
}

# Releases are signed with cosign (keyless, GitHub Actions OIDC); verify when cosign is installed.
function Test-Signature([string]$Base, [string]$Tmp) {
  if (-not (Get-Command cosign -ErrorAction SilentlyContinue)) {
    Write-Host 'cosign not found; skipping signature verification (checksum only)'
    return
  }
  Write-Host 'Verifying signature...'
  $bundle = Join-Path $Tmp 'checksums.txt.sigstore.json'
  Get-File "$Base/checksums.txt.sigstore.json" $bundle
  & cosign verify-blob (Join-Path $Tmp 'checksums.txt') `
    --bundle $bundle `
    --certificate-identity-regexp "^https://github.com/$Repo/" `
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
  # Native exit codes don't trigger $ErrorActionPreference on Windows PowerShell 5.1.
  if ($LASTEXITCODE -ne 0) { Fail 'Signature verification failed' }
}

function Add-ToUserPath([string]$Dir) {
  $normalized = $Dir.TrimEnd('\')
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  $entries = @($userPath -split ';' | Where-Object { $_ } | ForEach-Object { $_.TrimEnd('\') })
  if ($entries -notcontains $normalized) {
    $newPath = if ($userPath) { "$userPath;$Dir" } else { $Dir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "  Added $Dir to your user PATH; open a new terminal for it to take effect."
  }
  # Make the binary available in this session too.
  $sessionEntries = @($env:Path -split ';' | ForEach-Object { $_.TrimEnd('\') })
  if ($sessionEntries -notcontains $normalized) { $env:Path = "$Dir;$env:Path" }
}

$arch = Get-Arch
$base = if ($Version) { "https://github.com/$Repo/releases/download/$Version" } else { "https://github.com/$Repo/releases/latest/download" }
$asset = "${Binary}_windows_${arch}.zip"

$tmp = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
  Write-Host "Downloading $asset..."
  Get-File "$base/$asset" (Join-Path $tmp $asset)
  Get-File "$base/checksums.txt" (Join-Path $tmp 'checksums.txt')

  Test-Signature $base $tmp

  Write-Host 'Verifying checksum...'
  Test-Checksum (Join-Path $tmp 'checksums.txt') (Join-Path $tmp $asset) $asset

  $extract = Join-Path $tmp 'extract'
  Expand-Archive -Path (Join-Path $tmp $asset) -DestinationPath $extract -Force
  $exe = Join-Path $extract "$Binary.exe"
  if (-not (Test-Path $exe)) {
    Fail "$Binary.exe not found in archive. If Windows Security removed it, check Windows Security > Protection history, restore it, and re-run."
  }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  $target = Join-Path $InstallDir "$Binary.exe"
  try {
    # Windows locks running executables, so an upgrade fails while avochato is running.
    Copy-Item -Force $exe $target
  } catch {
    Fail "Could not write $target. Close any running avochato processes and re-run. ($($_.Exception.Message))"
  }

  Add-ToUserPath $InstallDir

  # The binary is not Authenticode-signed, so SmartScreen or Smart App Control may block its first run.
  try {
    & $target --version | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "exit code $LASTEXITCODE" }
  } catch {
    Fail "Installed $target, but running it failed: $($_.Exception.Message)`nIf Windows Security blocked it, check Windows Security > Protection history, allow $Binary.exe, and re-run."
  }

  Write-Host "avochato installed to $target"
  Write-Host '  Run: avochato login'
} finally {
  if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
}
