@echo off
REM Docker Build and Push Script for Redis Queue System (Windows)
REM Usage: scripts\docker-build.bat [tag] [registry]

setlocal enabledelayedexpansion

REM Default values
set DEFAULT_TAG=latest
set DEFAULT_REGISTRY=ghcr.io/harshaweb
set DEFAULT_IMAGE_NAME=queue

REM Parse command line arguments
set TAG=%1
if "%TAG%"=="" set TAG=%DEFAULT_TAG%

set REGISTRY=%2
if "%REGISTRY%"=="" set REGISTRY=%DEFAULT_REGISTRY%

set IMAGE_NAME=%3
if "%IMAGE_NAME%"=="" set IMAGE_NAME=%DEFAULT_IMAGE_NAME%

REM Full image name
set FULL_IMAGE_NAME=%REGISTRY%/%IMAGE_NAME%:%TAG%

echo ======================================
echo 🐳 Redis Queue System Docker Builder
echo ======================================
echo.

echo [INFO] Configuration:
echo   - Registry: %REGISTRY%
echo   - Image: %IMAGE_NAME%
echo   - Tag: %TAG%
echo   - Full image name: %FULL_IMAGE_NAME%
echo.

REM Check prerequisites
echo [INFO] Checking prerequisites...
docker version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not installed or not running
    exit /b 1
)
echo [SUCCESS] Prerequisites check passed

REM Get build context information
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
if "%GIT_COMMIT%"=="" set GIT_COMMIT=unknown

for /f %%i in ('git describe --tags --always 2^>nul') do set VERSION=%%i
if "%VERSION%"=="" set VERSION=dev

for /f "tokens=1-6 delims=: " %%a in ('echo %date% %time%') do (
    set BUILD_DATE=%%c-%%a-%%bT%%d:%%e:%%fZ
)

echo [INFO] Build context:
echo   - Git commit: %GIT_COMMIT%
echo   - Version: %VERSION%
echo   - Build date: %BUILD_DATE%

REM Build the image
echo [INFO] Building Docker image: %FULL_IMAGE_NAME%
docker build ^
    --build-arg GIT_COMMIT=%GIT_COMMIT% ^
    --build-arg BUILD_DATE=%BUILD_DATE% ^
    --build-arg VERSION=%VERSION% ^
    --tag %FULL_IMAGE_NAME% ^
    --tag %REGISTRY%/%IMAGE_NAME%:%GIT_COMMIT% ^
    .

if %errorlevel% neq 0 (
    echo [ERROR] Docker build failed
    exit /b 1
)

echo [SUCCESS] Docker image built successfully

REM Show image information
echo [INFO] Image information:
docker images %REGISTRY%/%IMAGE_NAME%

REM Ask for push confirmation
echo.
set /p PUSH_CONFIRM="Do you want to push the image to the registry? (y/N): "

if /i "%PUSH_CONFIRM%"=="y" (
    echo [INFO] Pushing image to registry: %REGISTRY%
    
    if "%REGISTRY:~0,8%"=="ghcr.io/" (
        echo [INFO] Detected GitHub Container Registry
        echo [INFO] Make sure you're logged in with: docker login ghcr.io -u USERNAME
    )
    
    echo [INFO] Pushing %FULL_IMAGE_NAME%...
    docker push %FULL_IMAGE_NAME%
    
    if %errorlevel% neq 0 (
        echo [ERROR] Failed to push main tag
        exit /b 1
    )
    
    echo [INFO] Pushing %REGISTRY%/%IMAGE_NAME%:%GIT_COMMIT%...
    docker push %REGISTRY%/%IMAGE_NAME%:%GIT_COMMIT%
    
    if %errorlevel% neq 0 (
        echo [ERROR] Failed to push commit tag
        exit /b 1
    )
    
    echo [SUCCESS] Images pushed successfully
    
    REM Cleanup
    echo [INFO] Cleaning up old images...
    docker image prune -f >nul 2>&1
    
    echo.
    echo [SUCCESS] 🎉 Docker build and push completed successfully!
    echo [INFO] You can now use the image with:
    echo   docker run -p 8080:8080 %FULL_IMAGE_NAME%
    echo.
    echo [INFO] Or deploy to Kubernetes:
    echo   kubectl set image deployment/queue-server queue-server=%FULL_IMAGE_NAME%
) else (
    echo [INFO] Image built but not pushed. You can push later with:
    echo   docker push %FULL_IMAGE_NAME%
)

echo.
echo [SUCCESS] Build process completed! 🚀

endlocal