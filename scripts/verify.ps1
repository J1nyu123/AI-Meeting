$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$repositoryRoot = Split-Path -Parent $projectRoot
$cacheRoot = Join-Path $projectRoot ".cache"
$goCache = Join-Path $cacheRoot "go-build"
$goTemp = Join-Path $cacheRoot "tmp"
$dockerConfig = Join-Path $cacheRoot "docker-config"
New-Item -ItemType Directory -Force $goCache, $goTemp, $dockerConfig | Out-Null
$env:GOCACHE = $goCache
$env:GOTMPDIR = $goTemp
$previousDockerConfig = $env:DOCKER_CONFIG
$env:DOCKER_CONFIG = $dockerConfig
Push-Location $projectRoot
try {
    Write-Output "[1/5] go build"
    go build ./...
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

    Write-Output "[2/5] go test"
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }

    Write-Output "[3/5] go vet"
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

    Write-Output "[4/5] docker compose config"
    docker compose config --quiet
    if ($LASTEXITCODE -ne 0) { throw "docker compose config failed with exit code $LASTEXITCODE" }

    Write-Output "[5/5] reference map"
    $referenceMap = Get-Content (Join-Path $projectRoot "docs/REFERENCE_MAP.md") -Raw -Encoding UTF8
    $references = [regex]::Matches($referenceMap, '`([^`]+\.(?:java|tsx?|xml|go))`')
    $missing = foreach ($reference in $references) {
        $relativePath = $reference.Groups[1].Value.Trim()
        $normalizedPath = $relativePath -replace '/', [IO.Path]::DirectorySeparatorChar
        $candidate = [IO.Path]::Combine($repositoryRoot, $normalizedPath)
        # File.Exists avoids a Windows PowerShell 5.1 Test-Path bug in Unicode workspaces.
        if (-not [IO.File]::Exists($candidate)) {
            $relativePath
        }
    }
    if ($missing) {
        throw "REFERENCE_MAP contains missing paths: $($missing -join ', ')"
    }
    Write-Output "Verified $($references.Count) traceable source/code paths"
} finally {
    Pop-Location
    $env:DOCKER_CONFIG = $previousDockerConfig
}
