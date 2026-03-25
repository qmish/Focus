# Kubernetes manifests (stage/prod)

Этот каталог содержит применимые манифесты для сред `stage` и `prod`.

## Структура

- `stage/focus.yaml` — namespace, deployments, services, HPA, ingress, network policies, secrets contract для stage.
- `prod/focus.yaml` — namespace, deployments, services, HPA (API/JVB), ingress TLS, network policies, secrets contract для prod.
- `hpa-api.yaml`, `hpa-frontend.yaml`, `hpa-jvb.yaml` — legacy HPA файлы (сохранены для обратной совместимости).
- `production.yaml` — исторический документ c примерами (не применяется напрямую).

## Применение

```bash
kubectl apply -f k8s/stage/focus.yaml
kubectl apply -f k8s/prod/focus.yaml
```

## Ротация секретов

Для ротации используется скрипт:

```powershell
.\scripts\rotate-k8s-secrets.ps1 -Environment stage
.\scripts\rotate-k8s-secrets.ps1 -Environment prod
```

После ротации выполнить restart деплойментов:

```bash
kubectl rollout restart deployment/api-go -n messenger-stage
kubectl rollout restart deployment/api-go -n messenger-prod
```
