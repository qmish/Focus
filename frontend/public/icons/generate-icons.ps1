Add-Type -AssemblyName System.Drawing

function New-Icon($size, $path, $maskable = $false) {
  $bmp = New-Object System.Drawing.Bitmap $size, $size
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias
  $bg = [System.Drawing.Color]::FromArgb(11, 19, 32)
  $g.Clear($bg)
  if (-not $maskable) {
    $accent = [System.Drawing.Color]::FromArgb(96, 165, 250)
    $brush = New-Object System.Drawing.SolidBrush $accent
    $r = [int]($size * 0.18)
    $g.FillEllipse($brush, $r, $r, $size - 2 * $r, $size - 2 * $r)
  } else {
    $accent = [System.Drawing.Color]::FromArgb(96, 165, 250)
    $brush = New-Object System.Drawing.SolidBrush $accent
    $r = [int]($size * 0.28)
    $g.FillEllipse($brush, $r, $r, $size - 2 * $r, $size - 2 * $r)
  }
  $fontSize = [int]($size * 0.45)
  $font = New-Object System.Drawing.Font 'Segoe UI', $fontSize, ([System.Drawing.FontStyle]::Bold)
  $textBrush = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::White)
  $sf = New-Object System.Drawing.StringFormat
  $sf.Alignment = [System.Drawing.StringAlignment]::Center
  $sf.LineAlignment = [System.Drawing.StringAlignment]::Center
  $rect = New-Object System.Drawing.RectangleF 0, 0, $size, $size
  $g.DrawString('F', $font, $textBrush, $rect, $sf)
  $g.Dispose()
  $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
  $bmp.Dispose()
}

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
New-Icon 192 (Join-Path $here 'icon-192.png') $false
New-Icon 512 (Join-Path $here 'icon-512.png') $false
New-Icon 512 (Join-Path $here 'icon-maskable.png') $true
Write-Host 'Icons generated.'
