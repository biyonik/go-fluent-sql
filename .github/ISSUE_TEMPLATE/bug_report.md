---
name: Bug Report
about: Create a report to help us improve
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

A clear and concise description of what the bug is.

## To Reproduce

Steps to reproduce the behavior:

1. Create query builder with '...'
2. Call method '...'
3. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Code Sample

```go
// Minimal code to reproduce the issue
qb := NewBuilder(db, grammar)
qb.Table("users").Where("id", "=", 1)
// ...
```

## Error Output

```
Paste the full error message here
```

## Environment

- **Go version**: [e.g., 1.22.0]
- **go-fluent-sql version**: [e.g., v0.1.0]
- **Database**: [e.g., MySQL 8.0, PostgreSQL 15]
- **OS**: [e.g., Ubuntu 22.04, macOS 14]

## Additional Context

Add any other context about the problem here.

## Possible Solution

If you have suggestions on how to fix the issue, please describe.
