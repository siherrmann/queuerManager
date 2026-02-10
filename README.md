# queuerManager

[![Go Reference](https://pkg.go.dev/badge/github.com/siherrmann/queuerManager.svg)](https://pkg.go.dev/github.com/siherrmann/queuerManager)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/siherrmann/queuerManager/blob/master/LICENSE)

A web-based management interface for the [queuer](https://github.com/siherrmann/queuer) job queueing system.

---

## 💡 Goal of this project

queuerManager provides an intuitive web interface to monitor and manage your queuer instances. It allows you to add jobs, monitor their execution, manage workers, configure tasks, and handle file uploads - all through a modern web UI built with htmx and Tailwind CSS.

---

## 🛠️ Installation

To integrate the queuerManager into your project, use the standard go get command:

```bash
go get github.com/siherrmann/queuerManager
```

The manager requires:

- A running postgres database (possibly with the timescaleDB extension, same usage as queuer)
- Node.js and npm for Tailwind CSS compilation

### Development Setup

```bash
# Install dependencies
make install

# Run the development server with hot reload
make run
```

This will start both the Go server with hot reload (using wgo and templ) and the Tailwind CSS watcher.

---

## 🚀 Getting Started

### Basic Usage

The simplest way to start the manager is:

```go
package main

import (
    "manager/helper"
)

func main() {
    helper.ManagerServer("3000", 1)
}
```

### Environment Variables

The manager uses the same database configuration as the queuer package:

```shell
QUEUER_DB_HOST=localhost
QUEUER_DB_PORT=5432
QUEUER_DB_DATABASE=postgres
QUEUER_DB_USERNAME=username
QUEUER_DB_PASSWORD=password1234
QUEUER_DB_SCHEMA=public
QUEUER_DB_WITH_TIMESCALE=true
```

Additional manager-specific variables:

```shell
QUEUER_MANAGER_PORT=3000
QUEUER_MANAGER_TASK_JSON=tasks_example.json  # Optional: Load tasks from JSON on startup
QUEUER_MANAGER_STORAGE_PATH=./uploads        # For local file storage
QUEUER_MANAGER_STORAGE_MODE=local            # local or s3
```

For S3 file storage, also configure:

```shell
S3_ENDPOINT=your-custom-endpoint
S3_REGION=eu-central-1
S3_BUCKET_NAME=bucket
S3_ACCESS_KEY_ID=access-key
S3_SECRET_ACCESS_KEY=secret-key
S3_USE_SSL=true
```

---

## ⭐ Features

### Job Management

- **Add Jobs**: Interactive web interface to add jobs with custom parameters
- **Job Monitoring**: View active jobs (queued, scheduled, running)
- **Job Archive**: Browse completed, cancelled, and failed jobs
- **Job Control**: Cancel individual or multiple jobs
- **Job Retry**: Re-add jobs from the archive with their original parameters

### Worker Management

- **Worker Overview**: Monitor all registered workers and their status
- **Worker Control**: Stop workers immediately or gracefully
- **Worker Health**: View worker heartbeat and connection status

### Task Management

- **Task Configuration**: Add, update, and delete task definitions
- **Task Import/Export**: Share task configurations between environments
- **Task Library**: Browse all available tasks with their parameters
- **JSON Import**: Bulk load tasks from a JSON file at startup

### File Management

- **File Upload**: Upload files for job processing
- **Storage Options**: Local filesystem or Amazon S3 support
- **File Browser**: View and manage uploaded files
- **Bulk Operations**: Delete multiple files at once

### System Monitoring

- **Database Connections**: Monitor active database connections
- **Health Check**: Built-in health check endpoint for monitoring
- **Real-time Updates**: Uses htmx for dynamic page updates without full reloads

### Security

- **CSRF Protection**: Built-in CSRF middleware for form submissions
- **Data Encryption**: Support for encrypting sensitive job data
- **Request Validation**: Input validation using the validator package

---

## 🖥️ Web Interface

Once started, the manager provides the following web views:

### Main Views

- **`/`** - Add Job: Interactive form to create new jobs
- **`/job`** - Job Details: View individual job information
- **`/jobs`** - Job List: Browse active jobs with pagination
- **`/jobArchive`** - Job Archive: View completed job history

### Worker Views

- **`/worker`** - Worker Details: View individual worker information
- **`/workers`** - Worker List: Browse all workers with their status

### Task Views

- **`/tasks`** - Task List: Browse all configured tasks
- **`/task`** - Task Details: View and edit task configuration

### File Views

- **`/files`** - File Browser: View and manage uploaded files
- **`/file`** - File Details: View individual file information

### API Endpoints

All views have corresponding REST API endpoints under `/api` for programmatic access:

- `/api/job/*` - Job operations
- `/api/worker/*` - Worker operations
- `/api/task/*` - Task operations
- `/api/file/*` - File operations
- `/api/connection/*` - Connection monitoring

---

## 📝 Task JSON Format

Tasks can be bulk imported from a JSON file. Example format:

```json
[
  {
    "key": "yourTask",
    "name": "Your task",
    "description": "Does task things.",
    "input_parameters": [
      {
        "Key": "source_text",
        "Type": "string",
        "Requirement": "min1"
      },
      {
        "Key": "source_language",
        "Type": "string",
        "Requirement": "equauto || equen || equde || equfr || eques || equit || equnl || equpt || equru || equzh"
      }
    ],
    "input_parameters_keyed": [
      {
        "Key": "batches",
        "Type": "int",
        "Requirement": "min1 || max1000"
      }
    ],
    "output_parameters": [
      {
        "Key": "output",
        "Type": "map",
        "Requirement": "min3",
        "InnerValidation": [
          {
            "Key": "text",
            "Type": "string",
            "Requirement": "-"
          },
          {
            "Key": "language",
            "Type": "string",
            "Requirement": "-"
          }
        ]
      }
    ]
  }
]
```

Set the `QUEUER_MANAGER_TASK_JSON` environment variable to automatically load tasks on startup.

---

## 🏗️ Architecture

The manager is built with:

- **[Echo](https://echo.labstack.com/)** - Fast HTTP framework
- **[templ](https://templ.guide/)** - Type-safe Go templating
- **[htmx](https://htmx.org/)** - Dynamic HTML without JavaScript
- **[Tailwind CSS](https://tailwindcss.com/)** - Utility-first CSS framework
- **[Hyperscript](https://hyperscript.org/)** - Frontend scripting
- **[queuer](https://github.com/siherrmann/queuer)** - Job queueing backend

The application follows a clean architecture with:

- **Handlers**: HTTP request handlers for views and API endpoints
- **Database**: Database access layer for tasks
- **Models**: Data structures and mappers
- **Middleware**: CSRF protection and request context
- **Upload**: File storage abstraction (local/S3)
- **View**: templ templates for UI components

---

## 🔧 Development

### Prerequisites

- Go 1.25+
- Node.js and npm
- PostgreSQL (with TimescaleDB extension)

### Development Commands

```bash
# Install dependencies
make install

# Run development server with hot reload
make run

# Clean up orphaned processes
make clean

# Run server only
make server

# Run Tailwind watcher only
make tailwind
```

### Building for Production

```bash
# Generate templ templates
templ generate

# Build Tailwind CSS
npx @tailwindcss/cli -i ./view/static/styles/index.css -o ./view/static/styles/output.css --minify

# Build the application
go build -o queuerManager .

# Run
./queuerManager
```
