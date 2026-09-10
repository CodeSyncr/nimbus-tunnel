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
| `TUNNEL_ADDR` | `:8090` | Listen address (plain HTTP; the edge terminates TLS) |
| `NIMBUS_CLOUD_URL` | `https://nimbusgo.space` | Cloud that authorizes CLI tokens |

`GET /healthz` returns `ok <active tunnels>`.

## Deploying on Coolify

1. **DNS (Cloudflare, DNS-only / grey cloud):** `A tunnel -> server IP` and `A *.tunnel -> server IP`. Keep them unproxied: Cloudflare's free certificate does not cover a second-level wildcard, and proxying would time out long-lived tunnels.
2. **Service:** new Dockerfile application from this repository, port `8090`, domains `https://tunnel.nimbusgo.space` and `https://*.tunnel.nimbusgo.space`, env `NIMBUS_CLOUD_URL=https://nimbusgo.space`.
3. **Wildcard certificate:** Let's Encrypt issues `*.tunnel.nimbusgo.space` only through a DNS challenge. Give the Coolify proxy a Cloudflare API token (Zone → DNS → Edit on the zone) as `CF_DNS_API_TOKEN` and a resolver:

   ```yaml
   certificatesResolvers:
     cloudflare:
       acme:
         email: you@example.com
         storage: /traefik/acme-cf.json
         dnsChallenge:
           provider: cloudflare
           resolvers: ["1.1.1.1:53", "1.0.0.1:53"]
   ```

   then on the service's router labels:

   ```
   traefik.http.routers.<name>.tls.certresolver=cloudflare
   traefik.http.routers.<name>.tls.domains[0].main=tunnel.nimbusgo.space
   traefik.http.routers.<name>.tls.domains[0].sans=*.tunnel.nimbusgo.space
   ```

4. Remove any response or idle timeouts on that route; tunnels and streamed responses are long-lived.

Verify with `curl https://tunnel.nimbusgo.space/healthz`, then `nimbus login` and `nimbus expose` from any project.

## Running locally

```sh
go run . # listens on :8090
NIMBUS_TUNNEL_URL=http://localhost:8090 nimbus expose 3333
```

Local tunnels resolve only if `*.tunnel.nimbusgo.space` (or your `TUNNEL_DOMAIN`) points at your machine; for a quick check use `curl -H 'Host: <name>.tunnel.nimbusgo.space' http://localhost:8090/`.
