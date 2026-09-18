param(
    [string]$Version = $(if ($env:REAPER_VERSION) { $env:REAPER_VERSION } else { "latest" }),
    [string]$InstallDir = $(if ($env:REAPER_INSTALL_DIR) { $env:REAPER_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\reaper" })
)

$ErrorActionPreference = "Stop"
$repo = "Eliran-Turgeman/repear"

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "reaper installer: 32-bit Windows is not supported"
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "reaper installer: unsupported CPU architecture: $env:PROCESSOR_ARCHITECTURE" }
}

if ($Version -eq "latest") {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $tag = $release.tag_name
} elseif ($Version.StartsWith("v")) {
    $tag = $Version
} else {
    $tag = "v$Version"
}

$releaseVersion = $tag.TrimStart("v")
if (-not $releaseVersion) {
    throw "reaper installer: could not determine the release version"
}

$asset = "reaper_${releaseVersion}_windows_${arch}.zip"
$baseUrl = "https://github.com/$repo/releases/download/$tag"
$tempDir = Join-Path ([IO.Path]::GetTempPath()) "reaper-$([guid]::NewGuid())"

try {
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    $archive = Join-Path $tempDir $asset
    $checksums = Join-Path $tempDir "checksums.txt"

    Write-Host "Downloading Reaper $releaseVersion for windows/$arch..."
    Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $archive
    Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksums

    $checksumLine = Get-Content $checksums | Where-Object { $_ -match "\s$([regex]::Escape($asset))$" }
    if (-not $checksumLine) {
        throw "reaper installer: $asset is missing from checksums.txt"
    }

    $expected = ($checksumLine -split "\s+")[0]
    $actual = (Get-FileHash -Algorithm SHA256 -Path $archive).Hash
    if ($actual -ne $expected) {
        throw "reaper installer: checksum verification failed"
    }

    Expand-Archive -Path $archive -DestinationPath $tempDir
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Force -Path (Join-Path $tempDir "reaper.exe") -Destination (Join-Path $InstallDir "reaper.exe")

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @($userPath -split ";" | Where-Object { $_ })
    if ($InstallDir -notin $pathEntries) {
        $newPath = (@($pathEntries) + $InstallDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "Added $InstallDir to your user PATH. Open a new terminal to use it."
    }

    Write-Host "Installed Reaper $releaseVersion to $(Join-Path $InstallDir 'reaper.exe')"
} finally {
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
}
