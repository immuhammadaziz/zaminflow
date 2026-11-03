#!/bin/sh
set -eu

if [ -n "${POSTGRES_USER_FILE:-}" ] || [ -n "${POSTGRES_PASSWORD_FILE:-}" ]; then
    : "${POSTGRES_USER_FILE:?POSTGRES_USER_FILE is required with POSTGRES_PASSWORD_FILE}"
    : "${POSTGRES_PASSWORD_FILE:?POSTGRES_PASSWORD_FILE is required with POSTGRES_USER_FILE}"

    if [ ! -r "$POSTGRES_USER_FILE" ] || [ ! -r "$POSTGRES_PASSWORD_FILE" ]; then
        echo "PostgreSQL credential files are not readable" >&2
        exit 1
    fi

    POSTGRES_USER="$(cat "$POSTGRES_USER_FILE")"
    if [ -z "$POSTGRES_USER" ]; then
        echo "POSTGRES_USER_FILE must not be empty" >&2
        exit 1
    fi
    export POSTGRES_USER
    unset POSTGRES_USER_FILE

    install -d -m 0700 -o postgres -g postgres /run/postgres-secrets
    install -m 0400 -o postgres -g postgres "$POSTGRES_PASSWORD_FILE" /run/postgres-secrets/password
    export POSTGRES_PASSWORD_FILE=/run/postgres-secrets/password
elif [ "${APP_ENV:-development}" = "production" ]; then
    echo "Production requires POSTGRES_USER_FILE and POSTGRES_PASSWORD_FILE" >&2
    exit 1
fi

chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql
exec su-exec postgres /usr/local/bin/docker-entrypoint.sh "$@"
