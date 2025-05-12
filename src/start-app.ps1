param (
    [Parameter(Position=0)]
    [string]$Mode
)

Write-Host "Little Alchemy Web - Deployment Helper" -ForegroundColor Cyan

switch ($Mode) {
    "local" {
        Write-Host "Starting application in LOCAL mode..." -ForegroundColor Green
        
        # First stop any existing containers
        docker-compose down
        
        # Build with local config
        docker-compose build --no-cache
        
        # Start containers
        docker-compose up -d
        
        Write-Host "Application started in LOCAL mode!" -ForegroundColor Green
        Write-Host "Frontend: http://localhost:3000" -ForegroundColor Yellow
        Write-Host "Backend: http://localhost:8080" -ForegroundColor Yellow
    }    "production" {
        Write-Host "Starting application in PRODUCTION mode..." -ForegroundColor Blue
        
        # First stop any existing containers
        docker-compose down
        
        # Start containers using production config
        docker-compose -f docker-compose.production.yml up -d
        
        Write-Host "Application started in PRODUCTION mode!" -ForegroundColor Blue
        Write-Host "Frontend: http://localhost:3000" -ForegroundColor Yellow
        Write-Host "Backend: https://tubes2stima-production.up.railway.app" -ForegroundColor Yellow
    }
    default {
        Write-Host "Usage: .\start-app.ps1 [local|production]" -ForegroundColor White
        Write-Host ""
        Write-Host "local      - Run with local backend (http://localhost:8080)" -ForegroundColor Gray
        Write-Host "production - Run with production backend (https://tubes2stima-production.up.railway.app)" -ForegroundColor Gray
    }
}
