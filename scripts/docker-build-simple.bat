@echo off
REM Simple Docker Build Script for Local Development
REM Usage: scripts\docker-build-simple.bat

echo ======================================
echo 🐳 Building Redis Queue Docker Image
echo ======================================
echo.

set IMAGE_NAME=redis-queue-local
set TAG=latest

echo [INFO] Building image: %IMAGE_NAME%:%TAG%

REM Build using the simple Dockerfile
docker build -f Dockerfile.simple -t %IMAGE_NAME%:%TAG% .

if %errorlevel% neq 0 (
    echo [ERROR] Docker build failed
    exit /b 1
)

echo [SUCCESS] Docker image built successfully!

REM Show the built image
echo [INFO] Image details:
docker images %IMAGE_NAME%

echo.
echo [INFO] You can now run the image with:
echo   docker run -p 8080:8080 -p 9090:9090 %IMAGE_NAME%:%TAG%
echo.
echo [INFO] Or use with docker-compose:
echo   docker-compose -f docker-compose.build.yml up

REM Test if the image can start
echo.
set /p TEST_RUN="Do you want to test run the container? (y/N): "
if /i "%TEST_RUN%"=="y" (
    echo [INFO] Testing container startup...
    docker run --rm -d --name test-queue -p 8080:8080 %IMAGE_NAME%:%TAG%
    if %errorlevel% equ 0 (
        echo [SUCCESS] Container started successfully!
        timeout /t 5 >nul
        echo [INFO] Stopping test container...
        docker stop test-queue >nul 2>&1
        echo [SUCCESS] Test completed successfully!
    ) else (
        echo [ERROR] Container failed to start
    )
)

echo.
echo [SUCCESS] Build process completed! 🚀