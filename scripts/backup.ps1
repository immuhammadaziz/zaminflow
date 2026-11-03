param(
    [string]$OutputDirectory = (Join-Path (Split-Path -Parent $PSScriptRoot) "backups")
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root ".env.production"
$composeFile = Join-Path $root "deploy\docker-compose.prod.yml"
if (-not (Test-Path -LiteralPath $envFile -PathType Leaf)) {
    throw "Production environment file not found: $envFile"
}

$composeArgs = @("compose", "--env-file", $envFile, "-f", $composeFile)
$containerId = & docker @composeArgs ps -q postgres
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($containerId)) {
    throw "Could not locate the running production PostgreSQL container."
}
$databaseUser = & docker exec $containerId printenv POSTGRES_USER
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($databaseUser)) {
    throw "Could not read the PostgreSQL username from the database container."
}
$databaseName = & docker exec $containerId printenv POSTGRES_DB
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($databaseName)) {
    throw "Could not read the PostgreSQL database name from the database container."
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$backupName = "zaminflow-$([DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')).dump"
$backupPath = Join-Path $OutputDirectory $backupName
$containerPath = "/tmp/zaminflow-backup-$([guid]::NewGuid().ToString('N')).dump"
try {
    & docker exec $containerId pg_dump --format=custom --no-owner --no-acl --file=$containerPath --username=$databaseUser --dbname=$databaseName
    if ($LASTEXITCODE -ne 0) {
        throw "pg_dump failed."
    }
    & docker exec $containerId pg_restore --list $containerPath *> $null
    if ($LASTEXITCODE -ne 0) {
        throw "The generated PostgreSQL archive failed its integrity listing check."
    }
    & docker cp "$($containerId):$containerPath" $backupPath
    if ($LASTEXITCODE -ne 0) {
        throw "Could not copy the verified archive to $backupPath."
    }
    $hash = Get-FileHash -LiteralPath $backupPath -Algorithm SHA256
    "{0}  {1}" -f $hash.Hash, $backupName | Set-Content -LiteralPath "$backupPath.sha256" -Encoding ASCII
    Write-Output "Backup created and verified: $backupPath"
    Write-Output "SHA-256: $($hash.Hash)"
}
finally {
    & docker exec $containerId rm -f $containerPath *> $null
}
