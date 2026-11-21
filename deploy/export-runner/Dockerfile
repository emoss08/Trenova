# The data export runner for Managed Trenova: a one-shot job that restores an
# instance's database from its backups into its own volume, dumps it, copies
# the instance's documents, and stores one archive in the export's bucket. It
# never connects to the instance. The control plane starts it; the decision
# behind it is 0026 in the control plane's documents.
#
# It is built FROM the Trenova PostgreSQL image of the release it exports,
# so the server that replays the backup and pg_dump are that release's
# PostgreSQL, with its extensions (PostGIS, pg_cron, pgvector). Every release
# tags it with its version; the control plane runs <repository>:<version> for
# an instance on that version. A release rebuilds it when this directory or
# the PostgreSQL image changed, FROM the release's PostgreSQL image by digest,
# and otherwise tags the previous digest (.github/workflows/release.yml,
# deploy/release/images.yaml).
#
# By hand:
#
#   docker build -f deploy/export-runner/Dockerfile \
#     --build-arg POSTGRES_IMAGE=<registry>/trenova/postgres:<version> \
#     -t <registry>/trenova/export-runner:<version> deploy/export-runner
ARG POSTGRES_IMAGE
FROM ${POSTGRES_IMAGE}

# Barman Cloud reads the backups CloudNativePG's Barman Cloud plugin writes;
# the same major version as the plugin's own image.
ARG BARMAN_VERSION=3.14.0

USER root
# psycopg2, which Barman requires, is compiled against the image's own libpq;
# the compilers go once it is built. Barman still imports distutils, which
# Python dropped in 3.12: setuptools brings it back for the runner's venv.
RUN apk add --no-cache bash coreutils curl jq python3 su-exec tar && \
    apk add --no-cache --virtual .build-deps build-base python3-dev && \
    python3 -m venv /opt/runner && \
    /opt/runner/bin/pip install --no-cache-dir "barman[cloud]==${BARMAN_VERSION}" setuptools && \
    apk del .build-deps && \
    /opt/runner/bin/barman-cloud-restore --version && \
    /opt/runner/bin/python3 -c "import boto3"

COPY --chmod=0755 export.sh test.sh /opt/runner/
COPY --chmod=0644 lib.sh s3.py README.txt /opt/runner/
# The build fails unless the scripts parse and the recovery wait works
# against this image's own PostgreSQL (test.sh).
RUN bash -n /opt/runner/export.sh && bash -n /opt/runner/lib.sh && \
    /opt/runner/bin/python3 -m py_compile /opt/runner/s3.py && \
    cd /tmp && su-exec postgres /opt/runner/test.sh
ENV PATH=/opt/runner/bin:$PATH
WORKDIR /tmp
ENTRYPOINT ["/opt/runner/export.sh"]
