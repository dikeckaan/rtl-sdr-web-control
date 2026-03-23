#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
docker compose down || true
docker rm -f openwebrx >/dev/null 2>&1 || true
