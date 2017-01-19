#!/bin/bash
set -e

if test -n "${GIT_SYNC_KEY}"; then
  mkdir -p /root/.ssh
  echo -e "${GIT_SYNC_KEY}" > /root/.ssh/id_rsa
  chmod 0600 /root/.ssh/id_rsa
fi

echo 'StrictHostKeyChecking no' >> /root/.ssh/config

exec /go/bin/git-sync "$@"