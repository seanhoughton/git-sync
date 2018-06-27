#!/bin/sh
set -e

mkdir -p /root/.ssh

if test -n "${GIT_SYNC_KEY}"; then
  echo -e "${GIT_SYNC_KEY}" > /root/.ssh/id_rsa
  chmod 0600 /root/.ssh/id_rsa
fi

echo 'StrictHostKeyChecking no' >> /root/.ssh/config

if [ "$1" = "server" ]; then
  shift
  exec /go/bin/git-sync "$@"
fi

exec "$@"
