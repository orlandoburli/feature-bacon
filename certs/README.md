# Development Certificates

This directory holds mTLS certificates for local development. These are **not** for production use.

A generation script will be added in Phase 4 when the first gRPC persistence module is implemented.

## Expected files (after generation)

```
certs/
├── ca.crt           # Certificate Authority
├── ca.key           # CA private key
├── core.crt         # bacon-core client certificate
├── core.key         # bacon-core client private key
├── module.crt       # Module server certificate
└── module.key       # Module server private key
```

All certificates are signed by the same CA. Both core and modules validate each other's certificates against this CA (mutual TLS).
