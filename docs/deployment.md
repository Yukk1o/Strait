# 部署指南

## Docker

### 构建镜像

```bash
docker build -t strait .
```

### 运行

```bash
docker run -d \
  -p 8080:8080 \
  -e DEEPSEEK_API_KEY=your-key \
  -v $(pwd)/configs:/app/configs \
  strait
```

### Docker Compose

```bash
# 设置环境变量
export DEEPSEEK_API_KEY=your-key

# 启动
docker-compose up -d
```

`docker-compose.yml` 会自动挂载 `configs/` 目录并注入环境变量。

---

## Kubernetes

### 1. 创建 Secret（存放 API Key）

参考 `k8s/secret.example.yaml`：

```bash
kubectl create secret generic strait-secret \
  --from-literal=DEEPSEEK_API_KEY=your-key
```

### 2. 创建 ConfigMap（挂载配置文件）

```bash
kubectl create configmap strait-config \
  --from-file=configs/
```

### 3. 部署

```bash
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
```

### 4. 验证

```bash
# 检查 Pod 状态
kubectl get pods -l app=strait

# 健康检查
kubectl port-forward svc/strait 8080:8080
curl http://localhost:8080/health
```

### 探针说明

| 探针 | 路径 | 说明 |
|------|------|------|
| liveness | `/health` | 存活检测，失败则重启容器 |
| readiness | `/ready` | 就绪检测，失败则从 Service 摘除 |

---

## 监控

### Prometheus Metrics

服务暴露 `/metrics` 端点，返回 Prometheus 格式的请求计数：

```bash
curl http://localhost:8080/metrics
```

---

## 配置热重载

修改 `configs/` 目录下的 YAML 文件后自动重载，无需重启：

- `providers.yaml` / `routes.yaml` → 路由重载
- `plugins.yaml` → 插件重载
