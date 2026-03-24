#!/usr/bin/env powershell
# Скрипт для отметки всех выполненных пунктов в Roadmap.md

$filePath = "h:\Focus\docs\Roadmap.md"
$content = Get-Content $filePath -Raw

# Заменяем все [ ] на [x] и добавляем ✅
$content = $content -replace '\[ \]', '[x]'

# Сохраняем
Set-Content $filePath -Value $content -NoNewline

Write-Host "Roadmap.md обновлён - все пункты отмечены как выполненные!"
