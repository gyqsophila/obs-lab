
# obs-lab (Cloud-Native Observability Lab)

一键在 K8s（kind/minikube/云上）拉起：
- **OpenTelemetry Collector**（OTLP 接入、Batch、Tail Sampling）
- **kube-prometheus-stack**（Prometheus + Alertmanager + Grafana）
- **Tempo**（分布式追踪）
- **Loki + Promtail**（日志采集与查询）
- **demo-go**（带 OTel Trace + Prometheus 指标 + 结构化日志，支持 Trace↔Log 互跳）

> 目标：打通 **Metrics ↔ Traces ↔ Logs** 的最小可用闭环，附 SLO 样例与告警规则。

---

## 0. 先决条件
- kubectl >= 1.24
- Helm >= 3.11
- 可选：kind 或 minikube（或任何可用 K8s 集群）
- Go >= 1.22 用于本地构建 demo-go（也可用预构建镜像 ghcr.io/chanlabs/obs-demo-go:latest *占位*）

> 你也可以直接 `helm template` 渲染并查看所有 YAML。

## 1. 快速开始（以 kind 为例）

```bash
# 1) 创建集群（若已有集群可跳过）
kind create cluster --name obs-lab

# 2) 添加 helm 仓库
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add opentelemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update

# 3) 安装依赖组件（监控栈 + Tempo + Loki/Promtail + OTel Collector）
helm upgrade --install monitoring prometheus-community/kube-prometheus-stack   -n monitoring --create-namespace   -f deploy/helm/values-kube-prom-stack.yaml

helm upgrade --install tempo grafana/tempo   -n monitoring   -f deploy/helm/values-tempo.yaml

helm upgrade --install loki grafana/loki   -n monitoring   -f deploy/helm/values-loki.yaml

helm upgrade --install promtail grafana/promtail   -n monitoring   -f deploy/helm/values-promtail.yaml

helm upgrade --install otel-collector opentelemetry/opentelemetry-collector   -n monitoring   -f deploy/helm/values-otel-collector.yaml

# 4) 部署 demo-go
helm upgrade --install demo-go deploy/helm/demo-go   -n demo --create-namespace   -f deploy/helm/demo-go/values.yaml

# 5) 端口转发 Grafana（初始账号 admin/prom-operator）
kubectl -n monitoring port-forward svc/monitoring-grafana 3000:80
# 浏览器打开 http://localhost:3000
# 数据源已自动配置（Prometheus / Loki / Tempo）
```

> **注意**：首次启动需等待 1–3 分钟，直到相关 Pod 就绪。

## 2. 架构图（文字）
```
demo-go (OTLP Traces + Prometheus Metrics + JSON Logs[trace_id])
   ├─ Traces → OTel Collector → Tempo
   ├─ Metrics → ServiceMonitor → Prometheus（Grafana 仪表盘 & Exemplars）
   └─ Logs → Promtail → Loki（LogQL 通过 trace_id/operation_name 聚合）
```

## 3. 功能点
- **Tail-based Sampling**：错误/慢请求优先保留（在 `values-otel-collector.yaml` 可配）。
- **Exemplars（实验性）**：直方图指标携带 trace exemplars（依赖客户端/抓取链配置）。
- **Trace ↔ Log 互跳**：日志打印 `trace_id`，Grafana Explore 可用。
- **SLO/告警**：示例规则见 `dashboards/slo/` 与 `deploy/helm/values-kube-prom-stack.yaml` 的 `additionalPrometheusRulesMap`。

## 4. 目录结构
```
/deploy
  /helm
    /demo-go                # demo 应用的 Helm Chart（Deployment/Service/ServiceMonitor）
    values-kube-prom-stack.yaml
    values-tempo.yaml
    values-loki.yaml
    values-promtail.yaml
    values-otel-collector.yaml
/apps
  /demo-go
    main.go                 # Gin/chi 二选一，这里用 chi
    go.mod / go.sum
    Dockerfile
/dashboards
  /grafana
    demo-overview.json      # RED + Exemplars
/slo
  slo-rules.yaml            # Sloth/Prometheus 规则样例
```

## 5. 验证
- 调用 demo 接口：
```bash
# 发送一些请求，包含部分错误/慢请求（通过 header 或 query 控制）
kubectl -n demo port-forward svc/demo-go 8080:80 &
for i in {1..200}; do
  curl -s "http://localhost:8080/hello?sleep_ms=$((RANDOM%200))" >/dev/null
done
curl -s "http://localhost:8080/boom" || true
```

- 在 Grafana：
  - `Explore → Tempo`：按 `service.name="demo-go"` 搜索 Trace；
  - `Explore → Loki`：`{app="demo-go"} |= "trace_id="`；
  - 打开仪表盘 **Demo Service Overview** 看 RED/Exemplars。

## 6. 常见问题
- Pod 未就绪：`kubectl get pods -A`；查看 `otel-collector`、`promtail`、`tempo` 日志。
- Tempo 查询不到 Trace：检查 `OTEL_EXPORTER_OTLP_ENDPOINT` 与 collector exporter 到 Tempo 的地址。
- Grafana 登录：默认 `admin/prom-operator`（kube-prometheus-stack 默认）。

## 7. 清理
```bash
helm -n demo uninstall demo-go || true
helm -n monitoring uninstall otel-collector promtail loki tempo monitoring || true
kubectl delete ns demo monitoring --ignore-not-found
kind delete cluster --name obs-lab || true
```

---

## 8. 许可证
MIT
