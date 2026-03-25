Write-Host "Starting Focus dev stack via docker compose..." -ForegroundColor Cyan
docker compose --env-file ".env" up -d --build

Write-Host ""
Write-Host "Dev stack is starting. Service URLs:" -ForegroundColor Green
Write-Host "  API:            http://localhost:8080"
Write-Host "  Keycloak:       http://localhost:8180"
Write-Host "  Frontend:       http://localhost:5173"
Write-Host "  Frontend Admin: http://localhost:5174"
Write-Host ""
Write-Host "Tail logs: docker compose logs -f" -ForegroundColor Yellow
