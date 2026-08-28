Write-Host "Checking Ollama status..." -ForegroundColor Cyan

# Check if Ollama is running
$ollamaRunning = Test-NetConnection -ComputerName 127.0.0.1 -Port 11434 -InformationLevel Quiet

if (-not $ollamaRunning) {
    Write-Host "Starting Ollama background service..." -ForegroundColor Yellow
    Start-Process "ollama" -ArgumentList "serve" -WindowStyle Hidden
    Start-Sleep -Seconds 3
}

Write-Host "Ensuring model qwen2.5-coder:3b is ready..." -ForegroundColor Cyan
ollama pull qwen2.5-coder:3b

Write-Host "Launching bruncher-AI..." -ForegroundColor Green
go run main.go