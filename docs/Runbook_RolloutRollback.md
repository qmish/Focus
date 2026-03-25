# Runbook: Rollout/Rollback (stage -> prod)

Документ фиксирует стратегию безопасного релиза в `stage` и `prod`.

## Стратегия rollout

1. **Quality gates в CI**
   - backend tests + lint;
   - frontend/frontend-admin tests;
   - e2e API smoke;
   - load smoke (k6);
   - security gates (`govulncheck`, `npm audit`).
2. **Deploy в stage**
   - обновить image tag в stage-манифестах/helm values;
   - выполнить `kubectl apply`/`helm upgrade` в namespace stage;
   - проверить `rollout status` и smoke-check.
3. **Validation в stage**
   - health/ready;
   - авторизация, базовый room/chat flow;
   - webhook/bot/admin smoke.
4. **Promote в prod**
   - использовать тот же image tag, что и в stage;
   - запуск в maintenance window;
   - контролируемый rollout (`maxUnavailable=0`, `maxSurge=1`).

## Команды rollout

```bash
# Проверка статуса
kubectl rollout status deployment/api-go -n messenger-stage
kubectl rollout status deployment/frontend -n messenger-stage
kubectl rollout status deployment/frontend-admin -n messenger-stage

# Продвижение в prod (пример)
kubectl set image deployment/api-go api-go=ghcr.io/qmish/focus-api:<TAG> -n messenger-prod
kubectl set image deployment/frontend frontend=ghcr.io/qmish/focus-frontend:<TAG> -n messenger-prod
kubectl set image deployment/frontend-admin frontend-admin=ghcr.io/qmish/focus-frontend-admin:<TAG> -n messenger-prod
```

## Критерии отката

- `5xx`/latency выше SLO после релиза;
- провал health/readiness;
- критические ошибки auth/websocket/webhook flow;
- регрессия основных пользовательских сценариев.

## Команды rollback

```bash
# Откат последней ревизии
kubectl rollout undo deployment/api-go -n messenger-prod
kubectl rollout undo deployment/frontend -n messenger-prod
kubectl rollout undo deployment/frontend-admin -n messenger-prod

# Откат до конкретной ревизии
kubectl rollout undo deployment/api-go -n messenger-prod --to-revision=2
kubectl rollout undo deployment/frontend -n messenger-prod --to-revision=2
kubectl rollout undo deployment/frontend-admin -n messenger-prod --to-revision=2
```

После отката:
- повторить smoke-check (`/health`, `/ready`, login, rooms/messages, admin stats);
- зафиксировать инцидент и root-cause в postmortem.

