# Contributing to Redis Queue System

First off, thank you for considering contributing to the Redis Queue System! 🎉

It's people like you that make this project a great tool for the community.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Process](#development-process)
- [Style Guidelines](#style-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)
- [Community](#community)

## 📜 Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## 🚀 Getting Started

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Redis 7.0+** - [Install Redis](https://redis.io/download)
- **Docker** (optional) - [Install Docker](https://docs.docker.com/get-docker/)
- **Git** - [Install Git](https://git-scm.com/downloads)

### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/Queue.git
   cd Queue
   ```

3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/harshaweb/Queue.git
   ```

4. **Install dependencies**:
   ```bash
   go mod download
   ```

5. **Start Redis** for development:
   ```bash
   # Using Docker
   docker run -d --name redis-dev -p 6379:6379 redis:7-alpine
   
   # Or using local Redis
   redis-server
   ```

6. **Run tests** to verify setup:
   ```bash
   go test ./...
   ```

7. **Test the applications**:
   ```bash
   # Test environment configuration
   go run examples/env-test/main.go
   
   # Test demo application  
   go run examples/demo/main.go
   
   # Build and test server
   go build ./cmd/queue-server
   go build ./cmd/queue-cli
   ```

## 🤝 How Can I Contribute?

### 🐛 Reporting Bugs

Before creating bug reports, please check [existing issues](https://github.com/harshaweb/Queue/issues) as you might find that the problem has already been reported.

When creating a bug report, use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment information
- Relevant logs/screenshots

### 💡 Suggesting Features

Feature requests are welcome! Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) and include:
- Clear problem description
- Proposed solution
- Use cases and benefits
- Implementation ideas (optional)

### 🔧 Code Contributions

1. **Check existing issues** for good first issues labeled `good first issue`
2. **Comment on the issue** to let others know you're working on it
3. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. **Make your changes** following our guidelines
5. **Add/update tests** for your changes
6. **Update documentation** if needed
7. **Commit your changes** with clear messages
8. **Push to your fork** and create a Pull Request

### 📚 Documentation

Documentation improvements are always welcome:
- Fix typos or unclear explanations
- Add examples and use cases
- Improve API documentation
- Write tutorials or guides
- Update outdated information

## 🛠️ Development Process

### Branch Strategy

- **`main`** - Production-ready code, protected branch
- **`develop`** - Integration branch for features (if used)
- **`feature/*`** - Feature branches
- **`bugfix/*`** - Bug fix branches
- **`hotfix/*`** - Critical fixes for production

### Pull Request Process

1. **Ensure tests pass**: `go test ./...`
2. **Update documentation** for any API changes
3. **Follow commit message format**: 
   ```
   type(scope): description
   
   - feat: add new feature
   - fix: bug fix
   - docs: documentation update
   - test: add or update tests
   - refactor: code refactoring
   - perf: performance improvement
   ```
4. **Fill out PR template** completely
5. **Wait for review** - be responsive to feedback
6. **Squash commits** if requested

### Review Process

All submissions require review:
- Maintainers will review PRs within 48 hours
- Address review feedback promptly
- Maintain respectful communication
- Be open to suggestions and changes

## 🎨 Style Guidelines

### Go Code Style

Follow standard Go conventions:
- **Format**: Use `gofmt` and `goimports`
- **Linting**: Use `golangci-lint run`
- **Naming**: Use Go naming conventions
- **Comments**: Document public APIs
- **Error Handling**: Always handle errors appropriately

```bash
# Format code
gofmt -w .
goimports -w .

# Run linters
golangci-lint run
```

### Code Organization

```
pkg/           # Library packages
cmd/           # Main applications
internal/      # Private application code
examples/      # Example applications
test/          # Test utilities and integration tests
docs/          # Documentation
deploy/        # Deployment configurations
scripts/       # Build and utility scripts
```

### API Design

- **RESTful**: Follow REST principles for HTTP APIs
- **Versioning**: Use `/api/v1/` for API versioning
- **Consistency**: Maintain consistent naming and structure
- **Documentation**: Document all APIs with OpenAPI/Swagger

### Configuration

- **Environment Variables**: Use `.env` files for configuration
- **Validation**: Validate all configuration inputs
- **Defaults**: Provide sensible defaults
- **Documentation**: Document all configuration options

## 🧪 Testing

### Test Types

1. **Unit Tests**: Test individual functions/methods
   ```bash
   go test ./pkg/...
   ```

2. **Integration Tests**: Test component interactions
   ```bash
   go test ./test/integration/...
   ```

3. **Benchmark Tests**: Performance testing
   ```bash
   go test -bench=. ./test/benchmark/...
   ```

4. **End-to-End Tests**: Full system testing
   ```bash
   go test ./test/e2e/...
   ```

### Test Guidelines

- **Coverage**: Aim for >80% test coverage
- **Isolation**: Tests should be independent
- **Reliability**: Tests should be deterministic
- **Speed**: Unit tests should be fast (<100ms)
- **Clarity**: Test names should describe what they test

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test package
go test ./pkg/queue/

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## 📖 Documentation

### Documentation Types

1. **Code Comments**: Document public APIs
2. **README Files**: Overview and quick start
3. **Technical Docs**: Detailed explanations in `/docs`
4. **Examples**: Working code examples
5. **API Docs**: Generated from code comments

### Documentation Guidelines

- **Clear**: Use simple, clear language
- **Complete**: Cover all features and options
- **Examples**: Provide working examples
- **Up-to-date**: Keep docs synchronized with code
- **Accessible**: Consider different skill levels

### Building Documentation

```bash
# Generate Go documentation
godoc -http=:6060

# Build API documentation (if using tools like swag)
swag init

# Test documentation examples
go test ./examples/...
```

## 🌟 Community

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Pull Requests**: Code contributions and reviews

### Getting Help

- Check the [documentation](docs/)
- Search [existing issues](https://github.com/harshaweb/Queue/issues)
- Ask in [discussions](https://github.com/harshaweb/Queue/discussions)
- Look at [examples](examples/)

### Recognition

Contributors are recognized in:
- [Contributors](https://github.com/harshaweb/Queue/graphs/contributors) page
- Release notes for significant contributions
- Special mentions for outstanding help

## 🎯 Good First Issues

Looking for a place to start? Check issues labeled:
- [`good first issue`](https://github.com/harshaweb/Queue/labels/good%20first%20issue) - Good for newcomers
- [`help wanted`](https://github.com/harshaweb/Queue/labels/help%20wanted) - Need community help
- [`documentation`](https://github.com/harshaweb/Queue/labels/documentation) - Documentation improvements

## ❓ Questions?

Don't hesitate to ask questions:
- Create a [discussion](https://github.com/harshaweb/Queue/discussions)
- Open a [question issue](https://github.com/harshaweb/Queue/issues/new?template=question.md)
- Reach out to maintainers in existing issues

## 🙏 Thank You!

Your contributions make this project better for everyone. Thank you for taking the time to contribute! 

---

**Happy coding!** 🚀