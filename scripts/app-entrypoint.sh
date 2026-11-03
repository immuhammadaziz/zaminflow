#!/bin/sh
set -eu

for secret_name in APP_SECRET METRICS_TOKEN DB_USER DB_PASSWORD SMTP_USER SMTP_PASSWORD ADMIN_PASSWORD; do
    file_variable="${secret_name}_FILE"
    eval "secret_path=\${${file_variable}:-}"
    if [ -z "$secret_path" ]; then
        continue
    fi
    case "$secret_path" in
        /run/secrets/*) ;;
        *)
            echo "$file_variable must point to a Docker secret under /run/secrets" >&2
            exit 1
            ;;
    esac
    if [ ! -r "$secret_path" ]; then
        echo "$file_variable is not readable" >&2
        exit 1
    fi
    secret_copy="/run/app-secrets/$secret_name"
    install -D -m 0400 -o 65532 -g 65532 "$secret_path" "$secret_copy"
    export "$file_variable=$secret_copy"
done

exec su-exec 65532:65532 "$@"
