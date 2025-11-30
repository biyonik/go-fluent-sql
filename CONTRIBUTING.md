# Contributing to go-fluent-sql

First off, thank you for considering contributing to go-fluent-sql! 🎉

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Process](#pull-request-process)
- [Coding Guidelines](#coding-guidelines)
- [Testing Guidelines](#testing-guidelines)
- [Commit Messages](#commit-messages)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/go-fluent-sql.git`
3. Add upstream remote: `git remote add upstream https://github.com/biyonik/go-fluent-sql.git`
4. Create a branch: `git checkout -b feature/your-feature-name`

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Make (optional, but recommended)
- Docker (optional, for integration tests)

### Setup

```bash
# Clone the repository
git clone https://github.com/biyonik/go-fluent-sql.git
cd go-fluent-sql

# Install dependencies
go mod download

# Install development tools
make install-tools

# Run tests to verify setup
make test
```

### Available Make Commands

```bash
make help          # Show all available commands
make test          # Run tests
make test-coverage # Run tests with coverage
make lint          # Run linter
make fmt           # Format code
make pre-commit    # Run all pre-commit checks
```

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing issues. When creating a bug report, include:

- **Clear title** describing the issue
- **Steps to reproduce** the behavior
- **Expected behavior** vs actual behavior
- **Code samples** if applicable
- **Go version** and OS information

### Suggesting Features

Feature requests are welcome! Please:

- Check if the feature has already been requested
- Provide a clear description of the feature
- Explain the use case and benefits
- Consider if it fits the project's scope

### Code Contributions

1. **Small changes**: Bug fixes, typos, documentation improvements
2. **Medium changes**: New features, refactoring
3. **Large changes**: Architectural changes, new modules

For medium/large changes, please open an issue first to discuss.

## Pull Request Process

1. **Update documentation** if you're changing functionality
2. **Add tests** for new features
3. **Ensure all tests pass**: `make test`
4. **Run linter**: `make lint`
5. **Update CHANGELOG.md** if applicable
6. **Fill out the PR template** completely

### PR Checklist

- [ ] Tests pass locally
- [ ] Linter passes
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] PR description explains changes

## Coding Guidelines

### Go Style

We follow the standard Go style guidelines:

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Specific Guidelines

```go
// ✅ DO: Use context as first parameter
func (qb *QueryBuilder) GetContext(ctx context.Context, dest any) error

// ❌ DON'T: Panic in library code
panic("something went wrong")

// ✅ DO: Return errors
return fmt.Errorf("query failed: %w", err)

// ✅ DO: Use meaningful variable names
userCount := len(users)

// ❌ DON'T: Use single letter names (except loops)
u := len(users)

// ✅ DO: Document exported functions
// GetContext executes the query and scans results into dest.
// It accepts a context for timeout and cancellation support.
func (qb *QueryBuilder) GetContext(ctx context.Context, dest any) error
```

### Error Handling

```go
// Use sentinel errors for expected conditions
var ErrNoRows = errors.New("fluentsql: no rows in result set")

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to compile query: %w", err)
}
```

## Testing Guidelines

### Test Structure

```go
func TestQueryBuilder_Where(t *testing.T) {
    // Arrange
    grammar := NewMySQLGrammar()
    qb := NewBuilder(nil, grammar)

    // Act
    qb.Table("users").Where("status", "=", "active")
    sql, args, err := qb.ToSQL()

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "SELECT * FROM `users` WHERE `status` = ?", sql)
    assert.Equal(t, []interface{}{"active"}, args)
}
```

### Test Naming

- `TestFunctionName_Scenario_ExpectedBehavior`
- Example: `TestQueryBuilder_Where_WithMaliciousInput_ReturnError`

### Coverage Requirements

- Minimum 70% coverage for new code
- Critical paths (security, data handling) should have higher coverage

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

### Examples

```
feat(builder): add WhereIn method with SQL injection protection

fix(grammar): resolve map iteration order non-determinism

docs(readme): add installation instructions

test(security): add SQL injection test cases
```

## Questions?

Feel free to open an issue with the `question` label or reach out to the maintainers.

Thank you for contributing! 🙏
