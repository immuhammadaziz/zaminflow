param(
    [Parameter(Mandatory = $true)]
    [string]$BackupPath
)

$ErrorActionPreference = "Stop"
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
$databaseName = "zaminflow_restore_test_$([guid]::NewGuid().ToString('N').Substring(0, 12))"
$containerPath = "/tmp/$databaseName.dump"
$databaseCreated = $false
try {
    & docker cp $backupPath "$($containerId):$containerPath"
    if ($LASTEXITCODE -ne 0) {
        throw "Could not copy the backup into the database container."
    }
    & docker exec $containerId createdb --username=$databaseUser $databaseName
    if ($LASTEXITCODE -ne 0) {
        throw "Could not create the isolated restore-test database."
    }
    $databaseCreated = $true
    & docker exec $containerId pg_restore --exit-on-error --no-owner --no-acl --username=$databaseUser --dbname=$databaseName $containerPath
    if ($LASTEXITCODE -ne 0) {
        throw "pg_restore failed against the isolated restore-test database."
    }
    $schemaCheck = & docker exec $containerId psql --no-psqlrc --tuples-only --no-align --username=$databaseUser --dbname=$databaseName --command "SELECT to_regclass('public.users') IS NOT NULL AND to_regclass('public.orders') IS NOT NULL"
    if ($LASTEXITCODE -ne 0 -or $schemaCheck.Trim() -ne "t") {
        throw "Restored database did not contain the expected application schema."
    }
    Write-Output "Restore verification passed in isolated database $databaseName."
}
finally {
    if ($databaseCreated) {
        & docker exec $containerId dropdb --if-exists --force --username=$databaseUser $databaseName *> $null
    }
    & docker exec --user 0 $containerId rm -f $containerPath *> $null
}
