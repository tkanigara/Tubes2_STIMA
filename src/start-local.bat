@echo off
echo Little Alchemy Web - Start Local Environment
echo.
echo Building and starting Docker containers...
cd %~dp0
docker-compose down
docker-compose build --no-cache
docker-compose up -d
echo.
echo Local environment ready!
echo.
echo Frontend: http://localhost:3000
echo Backend: http://localhost:8080
echo.
echo Press any key to exit...
pause > nul
