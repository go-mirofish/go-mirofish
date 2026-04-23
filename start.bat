@echo off
setlocal enabledelayedexpansion

set "ROOT_DIR=%~dp0"
if "%ROOT_DIR:~-1%"=="\" set "ROOT_DIR=%ROOT_DIR:~0,-1%"

set "BACKEND_DIR=%ROOT_DIR%\backend"
set "BACKEND_VENV=%BACKEND_DIR%\.venv"
set "GATEWAY_BIN=%ROOT_DIR%\gateway\bin\go-mirofish-gateway.exe"
set "FRONTEND_DIST_DIR=%ROOT_DIR%\frontend\dist"

if not "%BIND_HOST%"=="" goto bind_host_ready
set "BIND_HOST=127.0.0.1"
:bind_host_ready

if not "%BACKEND_PORT%"=="" goto backend_port_ready
set "BACKEND_PORT=5001"
:backend_port_ready

if not "%GATEWAY_PORT%"=="" goto gateway_port_ready
set "GATEWAY_PORT=3000"
:gateway_port_ready

if not exist "%GATEWAY_BIN%" (
  echo error: prebuilt gateway binary not found at %GATEWAY_BIN%
  exit /b 1
)

if not exist "%FRONTEND_DIST_DIR%\index.html" (
  echo error: frontend build output not found at %FRONTEND_DIST_DIR%\index.html
  exit /b 1
)

set "PYTHON_EXE=%BACKEND_VENV%\Scripts\python.exe"
if exist "%PYTHON_EXE%" goto python_ready

where py >nul 2>nul
if %errorlevel% neq 0 (
  echo error: Python launcher not found. Install Python 3.11.x. See docs\getting-started\installation.md
  exit /b 1
)

py -3.11 -m venv "%BACKEND_VENV%"
if %errorlevel% neq 0 (
  echo error: failed to create backend virtual environment
  exit /b 1
)

"%BACKEND_VENV%\Scripts\python.exe" -m pip install --upgrade pip >nul
"%BACKEND_VENV%\Scripts\pip.exe" install -r "%BACKEND_DIR%\requirements.txt"
if %errorlevel% neq 0 (
  echo error: failed to install backend requirements
  exit /b 1
)

:python_ready
start "go-mirofish-backend" /B cmd /c "cd /d %BACKEND_DIR% && \"%PYTHON_EXE%\" run.py"

set "BACKEND_READY="
for /l %%i in (1,1,60) do (
  powershell -NoProfile -Command "try { Invoke-WebRequest -UseBasicParsing http://127.0.0.1:%BACKEND_PORT%/health ^| Out-Null; exit 0 } catch { exit 1 }"
  if !errorlevel! equ 0 (
    set "BACKEND_READY=1"
    goto backend_ready
  )
  timeout /t 1 /nobreak >nul
)

if not defined BACKEND_READY (
  echo error: backend did not become healthy on port %BACKEND_PORT%
  exit /b 1
)

:backend_ready
set "BACKEND_URL=http://127.0.0.1:%BACKEND_PORT%"
set "FRONTEND_DIST_DIR=%FRONTEND_DIST_DIR%"
set "GATEWAY_BIND_HOST=%BIND_HOST%"
set "GATEWAY_PORT=%GATEWAY_PORT%"

"%GATEWAY_BIN%"
