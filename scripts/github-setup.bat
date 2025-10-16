@echo off
REM GitHub Repository Setup Script for Windows
REM This script provides commands to set up your GitHub repository properly

echo ========================================================
echo 🚀 Redis Queue System - GitHub Repository Setup
echo ========================================================
echo.

echo 📋 Manual Setup Required on GitHub.com:
echo.

echo 1. 📝 Repository Description:
echo    Go to: https://github.com/harshaweb/Queue/settings
echo    Set Description: Production-ready message queue system built on Redis Streams with Go SDK, REST/gRPC APIs, Kubernetes deployment, and enterprise-grade observability
echo    Set Website: https://github.com/harshaweb/Queue
echo.

echo 2. 🏷️ Repository Topics (click Add topics):
echo    redis, message-queue, golang, kubernetes, microservices
echo    redis-streams, distributed-systems, queue-system, go-sdk
echo    rest-api, grpc, prometheus, observability, docker
echo    helm-chart, high-availability, scalability, production-ready
echo    cloud-native, devops
echo.

echo 3. ⚙️ Repository Features (enable these):
echo    ✅ Issues
echo    ✅ Projects
echo    ✅ Wiki  
echo    ✅ Discussions
echo    ✅ Security advisories
echo.

echo 4. 🔒 Branch Protection (for main branch):
echo    Go to: https://github.com/harshaweb/Queue/settings/branches
echo    ✅ Require a pull request before merging
echo    ✅ Require status checks to pass before merging
echo    ✅ Require branches to be up to date before merging
echo    ✅ Include administrators
echo.

echo 5. 📦 Package Settings:
echo    Go to: https://github.com/harshaweb/Queue/settings
echo    Scroll to Features section
echo    ✅ Packages (for Docker images)
echo.

echo 6. 🏃 Actions Setup:
echo    Go to: https://github.com/harshaweb/Queue/actions
echo    Enable GitHub Actions (should auto-enable with workflow files)
echo.

gh --version >nul 2>&1
if %errorlevel% equ 0 (
    echo 🤖 GitHub CLI detected! You can run these commands:
    echo.
    echo # Set repository description
    echo gh repo edit harshaweb/Queue --description "Production-ready message queue system built on Redis Streams with Go SDK, REST/gRPC APIs, Kubernetes deployment, and enterprise-grade observability"
    echo.
    echo # Add topics (copy and run these commands one by one)
    echo gh repo edit harshaweb/Queue --add-topic redis
    echo gh repo edit harshaweb/Queue --add-topic message-queue
    echo gh repo edit harshaweb/Queue --add-topic golang
    echo gh repo edit harshaweb/Queue --add-topic kubernetes
    echo gh repo edit harshaweb/Queue --add-topic microservices
    echo gh repo edit harshaweb/Queue --add-topic redis-streams
    echo gh repo edit harshaweb/Queue --add-topic distributed-systems
    echo gh repo edit harshaweb/Queue --add-topic queue-system
    echo gh repo edit harshaweb/Queue --add-topic go-sdk
    echo gh repo edit harshaweb/Queue --add-topic rest-api
    echo gh repo edit harshaweb/Queue --add-topic grpc
    echo gh repo edit harshaweb/Queue --add-topic prometheus
    echo gh repo edit harshaweb/Queue --add-topic observability
    echo gh repo edit harshaweb/Queue --add-topic docker
    echo gh repo edit harshaweb/Queue --add-topic helm-chart
    echo gh repo edit harshaweb/Queue --add-topic high-availability
    echo gh repo edit harshaweb/Queue --add-topic scalability
    echo gh repo edit harshaweb/Queue --add-topic production-ready
    echo gh repo edit harshaweb/Queue --add-topic cloud-native
    echo gh repo edit harshaweb/Queue --add-topic devops
    echo.
    echo # Enable repository features
    echo gh repo edit harshaweb/Queue --enable-issues
    echo gh repo edit harshaweb/Queue --enable-projects  
    echo gh repo edit harshaweb/Queue --enable-wiki
    echo.
) else (
    echo 🤖 GitHub CLI not found. Install it from: https://cli.github.com/
    echo    Then run this script again for CLI commands.
)

echo.
echo 📝 Additional Recommendations:
echo.
echo 1. Create a README badge for build status
echo 2. Set up GitHub Sponsors (if desired)
echo 3. Create repository social media preview image
echo 4. Set up security policy (SECURITY.md)
echo 5. Configure repository insights and analytics
echo.

echo ✅ Once setup is complete, your repository will have:
echo    - Professional description and topics for discoverability
echo    - Proper branch protection rules
echo    - Issue and PR templates for contributors  
echo    - Contributing guidelines and code of conduct
echo    - Automated CI/CD pipeline with GitHub Actions
echo    - Docker image publishing to GitHub Container Registry
echo.

echo 🎉 Your Redis Queue System repository will be ready for the community!
echo.

pause