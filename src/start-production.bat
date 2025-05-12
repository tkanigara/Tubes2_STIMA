@echo off
echo Little Alchemy Web - Start Production Mode
echo.
echo This will build the application with the production backend URL
echo.
echo Building and starting Docker containers...
cd %~dp0

docker-compose down
docker-compose -f docker-compose.production.yml up -d

echo.
echo Production environment ready!
echo.
echo Frontend: http://localhost:3000
echo Backend: https://tubes2stima-production.up.railway.app
echo.
echo Press any key to exit...
pause > nul
