FROM postgres:16-alpine@sha256:0cefaabd882e42bcb2a2f3dc56d3ce4770560ba62fd7f197f1a2e008fd5731ab

USER root

RUN apk upgrade --no-cache \
    && apk add --no-cache su-exec \
    && rm -f /usr/local/bin/gosu \
    && chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql

COPY scripts/postgres-entrypoint.sh /usr/local/bin/postgres-secret-entrypoint.sh
RUN chmod 0755 /usr/local/bin/postgres-secret-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/postgres-secret-entrypoint.sh"]
CMD ["postgres"]
