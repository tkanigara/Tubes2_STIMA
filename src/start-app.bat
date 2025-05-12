@echo off
echo Little Alchemy Web - Deployment Helper

if "%1"=="local" (
    echo Starting application in LOCAL mode...
    docker-compose --profile local up -d
    echo Application started in LOCAL mode!
    echo Frontend: http://localhost:3000
    echo Backend: http://localhost:8080
) else if "%1"=="production" (
    echo Starting application in PRODUCTION mode...
    docker-compose --profile production up -d
    echo Application started in PRODUCTION mode!
    echo Frontend: http://localhost:3000
    echo Backend: https://tubes2stima-production.up.railway.app
) else (
    echo Usage: start-app.bat [local^|production]
    echo.
    echo local      - Run with local backend (http://localhost:8080)
    echo production - Run with production backend (https://tubes2stima-production.up.railway.app)
)
