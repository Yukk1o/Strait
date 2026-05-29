# Deployment Guide

English | [中文](../zh/deployment.md)

## Docker

### Build

```bash
docker build -t strait .
```

### Run

```bash
docker run -d \
  -p 8080:8080 \
  -e DEEPSEEK_API_KEY=your-key \
  -v $(pwd)/configs:/app/configs \
  strait
```

### Docker Compose

```bash
export DEEPSEEK_API_KEY=your-key
docker-compose up -d
```

`docker-compose.yml` auto-mounts `configs/` and injects environment variables.

---

## Kubernetes

### 1. Create Secret (API Key)

```bash
kubectl create secret generic strait-secret \
  --from-literal=DEEPSEEK_API_KEY=your-key
```

### 2. Create ConfigMap (config files)

```bash
kubectl create configmap strait-config \
  --from-file=configs/
```

### 3. Deploy

```bash
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
```

### 4. Verify

```bash
kubectl get pods -l app=strait
kubectl port-forward svc/strait 8080:8080
curl http://localhost:8080/health
```

### Probes

| Probe | Path | Description |
|-------|------|-------------|
| liveness | `/health` | Liveness check, restarts container on failure |
| readiness | `/ready` | Readiness check, removes from Service on failure |

---

## Monitoring

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `STRAIT_CORS_ORIGINS` | Comma-separated list of allowed CORS origins | All origins (dev only) |
| `STRAIT_BANNER` | Set to `false` to suppress startup banner | Shown |
| `*_API_KEY` | Provider API keys (e.g. `DEEPSEEK_API_KEY`) | — |

---

## Hot Reload

Modify YAML files under `configs/` — auto-reloaded without restart:

- `providers.yaml` / `routes.yaml` → route reload
- `plugins.yaml` → plugin reload
