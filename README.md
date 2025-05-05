# wol-go

A simple Wake-On-Lan service with a minimal HTTP front-end.

## Building

```bash
go build -o wol-go .
```

## Running

```bash
wol-go -config path/to/config.json
```

### Configuring

The service expects a config file in a JSON format, consisting of:

```js
{
    "address": "[<ip>]:<port> on which the service will listen",
    "broadcast": "The IPv4 broadcast of the network",
    "machines": [
        {
            "name": "Human-friendly identifier",
            "mac": "MAC address of the machine in hex format, separated by ':' or '-'",
            "ports": [/* A list of ports (uint16) on which to send the magic packet */]
        }
    ]
}
```
