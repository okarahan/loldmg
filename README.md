# loldmg

League of Legends fight analyzer.

## Web UI

The web app shows fights as a row of cards. Hover a card to see game time and duration; click it to open the damage meter.

```bash
RIOT_API_KEY=RGAPI-xxx go run ./cmd/server -addr=:8080
```

Open http://localhost:8080 and paste a match ID.

### Demo mode

No Riot API key needed — type `demo` as the match ID to load built-in sample data:

```bash
go run ./cmd/server -addr=:8080
```

Then enter `demo` and click Analyze.

### Local development with sample files

```bash
go run ./cmd/server -addr=:8080 -dev=internal/analyzer/testdata
```

## Docker

```bash
docker build -t ghcr.io/okarahan/loldmg:main .
docker run -p 8080:8080 -e RIOT_API_KEY=RGAPI-xxx ghcr.io/okarahan/loldmg:main
```

## Kubernetes

Manifests live in the homelab repo under `kubernetes/apps/loldmg`:

```bash
# In src/homelab
kubectl apply -f kubernetes/apps/loldmg/secret.example.yaml
kubectl apply -k kubernetes/apps/loldmg
```

The app is exposed at `https://loldmg.home.okarahan.arpa` (via Traefik Gateway API HTTPRoute). Add a private DNS entry or `/etc/hosts` line pointing `loldmg.home.okarahan.arpa` to your cluster ingress IP.

Flux ImageUpdateAutomation in `kubernetes/sources/loldmg-imageupdateautomation.yaml` keeps the deployment image pinned to the latest `main-<sha>` tag pushed by CI.

## GitHub Container Registry

Pushes happen automatically via `.github/workflows/docker.yml` on every push to `main` or tag `v*`. Images are pushed to `ghcr.io/okarahan/loldmg`.
