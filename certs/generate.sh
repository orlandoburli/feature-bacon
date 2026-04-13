#!/usr/bin/env bash
set -euo pipefail

# Generates development mTLS certificates for bacon-core and modules.
# Output: certs/dev/{ca,server,client}.{pem,key}
# Usage: ./certs/generate.sh

DIR="$(cd "$(dirname "$0")" && pwd)/dev"
rm -rf "$DIR"
mkdir -p "$DIR"

DAYS=365
SUBJ_CA="/CN=bacon-dev-ca"
SUBJ_SRV="/CN=bacon-module"
SUBJ_CLI="/CN=bacon-core"

echo "==> Generating CA"
openssl ecparam -genkey -name prime256v1 -out "$DIR/ca.key" 2>/dev/null
openssl req -new -x509 -key "$DIR/ca.key" -out "$DIR/ca.pem" \
  -days "$DAYS" -subj "$SUBJ_CA" 2>/dev/null

echo "==> Generating server cert (modules)"
openssl ecparam -genkey -name prime256v1 -out "$DIR/server.key" 2>/dev/null
openssl req -new -key "$DIR/server.key" -out "$DIR/server.csr" \
  -subj "$SUBJ_SRV" 2>/dev/null
openssl x509 -req -in "$DIR/server.csr" -CA "$DIR/ca.pem" -CAkey "$DIR/ca.key" \
  -CAcreateserial -out "$DIR/server.pem" -days "$DAYS" \
  -extfile <(printf "subjectAltName=DNS:localhost,DNS:module-postgres,DNS:module-kafka,DNS:module-redis,DNS:module-mongodb") 2>/dev/null

echo "==> Generating client cert (core)"
openssl ecparam -genkey -name prime256v1 -out "$DIR/client.key" 2>/dev/null
openssl req -new -key "$DIR/client.key" -out "$DIR/client.csr" \
  -subj "$SUBJ_CLI" 2>/dev/null
openssl x509 -req -in "$DIR/client.csr" -CA "$DIR/ca.pem" -CAkey "$DIR/ca.key" \
  -CAcreateserial -out "$DIR/client.pem" -days "$DAYS" \
  -extfile <(printf "subjectAltName=DNS:localhost,DNS:bacon-core") 2>/dev/null

rm -f "$DIR"/*.csr "$DIR"/*.srl

echo "==> Done. Certs in $DIR/"
ls -la "$DIR"
