#!/usr/bin/env bash
# Remove the throwaway Open WebUI container, its volume, and the .env that
# points the acceptance suite at it.
set -euo pipefail

CONTAINER=openwebui-testacc
VOLUME=openwebui-testacc-data

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
ENV_FILE="${REPO_ROOT}/.env"

docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
docker volume rm "${VOLUME}" >/dev/null 2>&1 || true
rm -f "${ENV_FILE}"

echo "Removed the ${CONTAINER} container, the ${VOLUME} volume, and ${ENV_FILE}."
