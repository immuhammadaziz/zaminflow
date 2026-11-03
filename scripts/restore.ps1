param(
    [Parameter(Mandatory = $true)]
    [string]$BackupPath,

    [Parameter(Mandatory = $true)]
    [string]$TargetDatabase,

    [switch]$Force
)

$ErrorActionPreference = "Stop"
if ($TargetDatabase -notmatch '^[A-Za-z_][A-Za-z0-9_]{0,62}$') {
    throw "TargetDatabase must be a PostgreSQL identifier containing only letters, digits, and underscores."
}
if (-not $Force) {
    $confirmation = Read-Host "This overwrites objects in '$TargetDatabase'. Type RESTORE to continue"
    if ($confirmation -cne "RESTORE") {
        throw "Restore cancelled."
    }
}
$backupPath = (Resolve-Path -LiteralPath $BackupPath).Path
$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root ".env.production"
$composeFile = Join-Path $root "deploy\docker-compose.prod.yml"
if (-not (Test-Path -LiteralPath $envFile -PathType Leaf)) {
    throw "Production environment file not found: $envFile"
}

$checksumPath = "$backupPath.sha256"
if (Test-Path -LiteralPath $checksumPath -PathType Leaf) {
    $expectedHash = ((Get-Content -LiteralPath $checksumPath -Raw).Trim() -split "\s+")[0]
    $actualHash = (Get-FileHash -LiteralPath $backupPath -Algorithm SHA256).Hash
    if ($expectedHash -ne $actualHash) {
        throw "Backup checksum validation failed."
    }
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
$containerPath = "/tmp/zaminflow-restore-$([guid]::NewGuid().ToString('N')).dump"
try {
    & docker cp $backupPath "$($containerId):$containerPath"
    if ($LASTEXITCODE -ne 0) {
        throw "Could not copy the backup into the database container."
    }
    & docker exec $containerId pg_restore --clean --if-exists --exit-on-error --no-owner --no-acl --username=$databaseUser --dbname=$TargetDatabase $containerPath
    if ($LASTEXITCODE -ne 0) {
        throw "pg_restore failed for target database '$TargetDatabase'."
    }
    Write-Output "Restore completed for database '$TargetDatabase'. Run restore-test.ps1 against a separate backup before relying on recovery."
}
finally {
    & docker exec --user 0 $containerId rm -f $containerPath *> $null
}
