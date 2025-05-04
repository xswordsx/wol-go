# wol-go

A simple Wake-On-Lan service with a minimal HTTP front-end.

## Building

```bash
go build -o wol-go .
```

## Configuring

There's currently only one option:
- `addr` - The address on which the service will start (default is `0.0.0.0:8080`)
