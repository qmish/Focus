param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("stage", "prod")]
    [string]$Environment
)

$namespace = if ($Environment -eq "stage") { "messenger-stage" } else { "messenger-prod" }

Write-Host "Rotating focus-secrets in namespace $namespace ..."

$dbPassword = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
$redisPassword = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
$jwtSecret = [Convert]::ToBase64String((1..64 | ForEach-Object { Get-Random -Maximum 256 }))

kubectl patch secret focus-secrets `
  -n $namespace `
  --type merge `
  -p "{`"stringData`":{`"db-password`":`"$dbPassword`",`"redis-password`":`"$redisPassword`",`"jwt-secret`":`"$jwtSecret`"}}" | Out-Null

Write-Host "Secrets rotated for $namespace."
Write-Host "Next steps:"
Write-Host "  kubectl rollout restart deployment/api-go -n $namespace"
Write-Host "  kubectl rollout restart deployment/frontend -n $namespace"
Write-Host "  kubectl rollout restart deployment/frontend-admin -n $namespace"
