#!/usr/bin/env bash
# Start a throwaway Open WebUI for the acceptance suite and write its
# credentials to .env. The suite creates, changes, and deletes real objects, so
# it must only ever point at a container that gets thrown away.
set -euo pipefail

CONTAINER=openwebui-testacc
VOLUME=openwebui-testacc-data
IMAGE=ghcr.io/open-webui/open-webui:v0.11.0
PORT=${OPENWEBUI_TESTACC_PORT:-8080}
ENDPOINT="http://localhost:${PORT}"

TEST_NAME="Terraform Acceptance"
TEST_EMAIL=testacc@localhost.local
TEST_PASSWORD=terraform-acc-test

HEALTH_TIMEOUT=${OPENWEBUI_TESTACC_TIMEOUT:-300}

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
ENV_FILE="${REPO_ROOT}/.env"

# Refuse to run when .env points anywhere but this machine. This is the
# mechanical protection against running the suite against a real instance.
if [ -f "${ENV_FILE}" ]; then
	current=$(sed -n 's/^OPENWEBUI_ENDPOINT=//p' "${ENV_FILE}" | tail -n 1 | tr -d '"'"'")
	case "${current}" in
	"" | http://localhost* | https://localhost* | http://127.0.0.1* | https://127.0.0.1*) ;;
	*)
		echo "Refusing to run: ${ENV_FILE} names ${current}." >&2
		echo "The acceptance suite must only run against a local throwaway container." >&2
		exit 1
		;;
	esac
fi

echo "Removing any previous ${CONTAINER} container and its volume"
docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
docker volume rm "${VOLUME}" >/dev/null 2>&1 || true

echo "Starting ${IMAGE} on port ${PORT}"
docker run -d \
	--name "${CONTAINER}" \
	-p "${PORT}:8080" \
	-v "${VOLUME}:/app/backend/data" \
	-e ENABLE_API_KEYS=true \
	-e WEBUI_SECRET_KEY=terraform-acc-test \
	-e WEBUI_AUTH=True \
	-e OFFLINE_MODE=true \
	-e RAG_EMBEDDING_ENGINE= \
	"${IMAGE}" >/dev/null

echo "Waiting up to ${HEALTH_TIMEOUT}s for ${ENDPOINT}/health"
deadline=$((SECONDS + HEALTH_TIMEOUT))
until curl -fsS "${ENDPOINT}/health" >/dev/null 2>&1; do
	if [ "$(docker inspect -f '{{.State.Running}}' "${CONTAINER}" 2>/dev/null)" != "true" ]; then
		echo "The container stopped before it answered /health. Last log lines:" >&2
		docker logs --tail 40 "${CONTAINER}" >&2 || true
		exit 1
	fi
	if [ "${SECONDS}" -ge "${deadline}" ]; then
		echo "${ENDPOINT}/health did not answer within ${HEALTH_TIMEOUT}s. Last log lines:" >&2
		docker logs --tail 40 "${CONTAINER}" >&2 || true
		exit 1
	fi
	sleep 2
done

# The database is empty, so this signup becomes the admin. Open WebUI does not
# gate the first user on ENABLE_SIGNUP.
echo "Creating the admin user ${TEST_EMAIL}"
signup=$(curl -fsS -X POST "${ENDPOINT}/api/v1/auths/signup" \
	-H 'Content-Type: application/json' \
	-d "$(jq -n --arg name "${TEST_NAME}" --arg email "${TEST_EMAIL}" --arg password "${TEST_PASSWORD}" \
		'{name: $name, email: $email, password: $password}')")

jwt=$(printf '%s' "${signup}" | jq -r '.token // empty')

if [ -z "${jwt}" ]; then
	echo "Signup returned no token." >&2
	exit 1
fi

# This first account is the primary admin, the one Open WebUI refuses to demote
# or delete. The user resource tests import it and assert those refusals.
admin_id=$(printf '%s' "${signup}" | jq -r '.id // empty')
admin_email=$(printf '%s' "${signup}" | jq -r '.email // empty')

if [ -z "${admin_id}" ] || [ -z "${admin_email}" ]; then
	echo "Signup returned no id or email for the primary admin." >&2
	exit 1
fi

# ENABLE_API_KEYS seeds auth.enable_api_keys on a fresh database. Without it
# this call answers 403.
echo "Creating an API key"
api_key=$(curl -fsS -X POST "${ENDPOINT}/api/v1/auths/api_key" \
	-H "Authorization: Bearer ${jwt}" |
	jq -r '.api_key // empty')

if [ -z "${api_key}" ]; then
	echo "The API key request returned no key." >&2
	exit 1
fi

# OPENWEBUI_TEST_CONFIG_IMPORT gates the config import and export tests, which
# rewrite the whole instance configuration. A throwaway container can afford it.
cat >"${ENV_FILE}" <<EOF
TF_ACC=1
OPENWEBUI_ENDPOINT=${ENDPOINT}
OPENWEBUI_TOKEN=${api_key}
OPENWEBUI_TEST_USER_EMAIL=${TEST_EMAIL}
OPENWEBUI_TEST_PRIMARY_ADMIN_ID=${admin_id}
OPENWEBUI_TEST_PRIMARY_ADMIN_EMAIL=${admin_email}
OPENWEBUI_TEST_CONFIG_IMPORT=1
EOF

echo "Wrote ${ENV_FILE}. Run 'make testacc' to test, 'make testacc-down' to clean up."
