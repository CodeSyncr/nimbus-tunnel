# nimbus-tunnel

The relay behind [`nimbus expose`](https://github.com/CodeSyncr/nimbus): it publishes a developer's local port on a public HTTPS URL such as `https://brisk-otter-3f9a.tunnel.nimbusgo.space`.

The protocol and proxy logic live in the framework's [`tunnel`](https://github.com/CodeSyncr/nimbus/tree/main/tunnel) package, shared with the CLI. This repository is only the deployable binary.

## How it works

1. `nimbus expose` dials `wss://tunnel.nimbusgo.space/connect` with the developer's Nimbus Cloud CLI token.
2. The relay asks Nimbus Cloud (`POST /api/v1/tunnel/authorize`) what that account may do: tunnel count, session length, custom subdomains.
3. Both ends wrap the WebSocket in a yamux session. Visitor requests on `<name>.tunnel.nimbusgo.space` are reverse-proxied through one stream each to the developer's machine, which proxies them to localhost.

HTTP/1.1, server-sent events and WebSocket upgrades pass through. Every response carries `X-Nimbus-Tunnel` for abuse tracing, and `Set-Cookie` headers scoped to a parent domain are dropped so an exposed app cannot plant cookies on the platform.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `TUNNEL_DOMAIN` | `tunnel.nimbusgo.space` | Suffix tunnels are published under |
| `TUNNEL_ADDR` | `:3000` | Listen address (plain HTTP; the edge terminates TLS) |
| `NIMBUS_CLOUD_URL` | `https://nimbusgo.space` | Cloud that authorizes CLI tokens |

`GET /healthz` returns `ok <active tunnels>`.

## Deploying on Coolify

Ready-made configuration lives in [`deploy/`](deploy/):

- [`deploy/coolify-proxy.yaml`](deploy/coolify-proxy.yaml): the server's Traefik proxy configuration (Servers → Proxy → Configuration) with a Cloudflare DNS-01 resolver for the wildcard certificate and no read timeout on long-lived connections. Paste your Cloudflare API token in place of `REPLACE_WITH_CLOUDFLARE_TOKEN`, save, restart the proxy.
- [`deploy/coolify-labels.txt`](deploy/coolify-labels.txt): the container labels for the resource (replace the generated Traefik block; Coolify's `Host(\`*.…\`)` rule never matches and its resolver cannot issue wildcards).

Steps:

1. **DNS (Cloudflare, DNS-only / grey cloud):** `A tunnel -> server IP` and `A *.tunnel -> server IP`. Keep them unproxied: Cloudflare's free certificate does not cover a second-level wildcard, and proxying would time out long-lived tunnels.
2. **Cloudflare API token:** profile → API Tokens → "Edit zone DNS" template, scoped to the nimbusgo.space zone.
3. **Proxy:** apply `deploy/coolify-proxy.yaml` with the token, restart the proxy.
4. **Resource:** Dockerfile application from this repository. Network: Ports Exposes `3000`, Port Mappings empty. Environment: `NIMBUS_CLOUD_URL=https://nimbusgo.space`. Container Labels: contents of `deploy/coolify-labels.txt`. Deploy.

Verify:

```sh
curl https://tunnel.nimbusgo.space/healthz          # ok 0
curl -I https://anything.tunnel.nimbusgo.space/      # valid certificate, 404 "Tunnel offline"
```

Then `nimbus login` and `nimbus expose` from any project.

## Running locally

```sh
go run . # listens on :3000
NIMBUS_TUNNEL_URL=http://localhost:3000 nimbus expose 3333
```

Local tunnels resolve only if `*.tunnel.nimbusgo.space` (or your `TUNNEL_DOMAIN`) points at your machine; for a quick check use `curl -H 'Host: <name>.tunnel.nimbusgo.space' http://localhost:3000/`.
