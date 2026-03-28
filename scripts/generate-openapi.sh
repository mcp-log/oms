#!/usr/bin/env bash
set -euo pipefail

SPEC="api/openapi/order-intake.yaml"
OUT_DIR="internal/orderintake/ports"
PACKAGE="ports"

echo "Generating OpenAPI types..."
oapi-codegen \
  -generate types \
  -package "${PACKAGE}" \
  -o "${OUT_DIR}/openapi_types.gen.go" \
  "${SPEC}"

echo "Generating OpenAPI Chi server..."
oapi-codegen \
  -generate chi-server \
  -package "${PACKAGE}" \
  -o "${OUT_DIR}/openapi_server.gen.go" \
  "${SPEC}"

echo "OpenAPI code generation complete."
