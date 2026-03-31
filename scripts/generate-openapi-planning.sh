#!/usr/bin/env bash
set -euo pipefail

SPEC="api/openapi/planning.yaml"
OUT_DIR="internal/planning/ports"
PACKAGE="ports"

echo "Generating Planning OpenAPI types..."
oapi-codegen \
  -generate types \
  -package "${PACKAGE}" \
  -o "${OUT_DIR}/openapi_types.gen.go" \
  "${SPEC}"

echo "Generating Planning OpenAPI Chi server..."
oapi-codegen \
  -generate chi-server \
  -package "${PACKAGE}" \
  -o "${OUT_DIR}/openapi_server.gen.go" \
  "${SPEC}"

echo "Planning OpenAPI code generation complete."
