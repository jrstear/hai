# Taskfile Usage

This directory uses [Task](https://taskfile.dev/) for common development tasks.

## Quick Start

```bash
# List all available tasks
task --list

# Or use the full path if task is not in PATH
/opt/homebrew/Cellar/go-task/3.45.5/bin/task --list
```

## Common Tasks

### Building

```bash
# Build the API server
task build

# Build the test-vcard utility
task build-test-vcard

# Build all binaries
task build-all
```

### Running

```bash
# Run the API server (builds first)
task run

# Run in development mode
task dev

# Run with custom port
PORT=3000 task run
```

### Testing

```bash
# Run Go tests
task test

# Test vCard import with sample file
task test-vcard

# Test vCard import with custom file
task test-vcard-custom FILE=path/to/contacts.vcf
```

### Code Quality

```bash
# Format code
task fmt

# Run linter
task lint

# Run all checks (fmt, lint, test)
task check

# Tidy dependencies
task tidy
```

### API Testing

```bash
# Check API health (server must be running)
task health

# List all contacts
task list-contacts

# Upload vCard file
task upload-vcard FILE=path/to/contacts.vcf
```

### Setup

```bash
# Initial setup (download dependencies)
task setup

# Clean build artifacts
task clean
```

## Environment Variables

The Taskfile uses these environment variables:

- `ELASTICSEARCH_URL` - Elasticsearch server URL (default: `http://localhost:9200`)
- `PORT` - API server port (default: `8080`)

You can override them:

```bash
ELASTICSEARCH_URL=http://localhost:9200 PORT=3000 task run
```

## Using the Specific Task Binary

If you need to use the specific task binary path:

```bash
/opt/homebrew/Cellar/go-task/3.45.5/bin/task <command>
```

Or create an alias in your shell:

```bash
alias task='/opt/homebrew/Cellar/go-task/3.45.5/bin/task'
```









