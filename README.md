# go-fluent-sql

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/biyonik/go-fluent-sql)](https://goreportcard.com/report/github.com/biyonik/go-fluent-sql)
[![codecov](https://codecov.io/gh/biyonik/go-fluent-sql/branch/main/graph/badge.svg)](https://codecov.io/gh/biyonik/go-fluent-sql)
[![GoDoc](https://godoc.org/github.com/biyonik/go-fluent-sql?status.svg)](https://pkg.go.dev/github.com/biyonik/go-fluent-sql)

A fluent, type-safe SQL query builder for Go with Laravel-inspired syntax.

## Features

- 🔗 **Fluent API** - Chain methods for readable queries
- 🛡️ **SQL Injection Protection** - Prepared statements & identifier validation
- 🎯 **Type Safety** - Compile-time checks where possible
- 🚀 **High Performance** - Minimal allocations, cached reflection
- 🔌 **Multi-Database** - MySQL, PostgreSQL (coming soon)
- 📦 **Zero Config** - Works out of the box
- 🧪 **Well Tested** - Comprehensive test coverage

## Installation

```bash
go get github.com/biyonik/go-fluent-sql
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    fluentsql "github.com/biyonik/go-fluent-sql"
)

func main() {
    // Connect to database
    db, err := fluentsql.Connect("user:password@tcp(localhost:3306)/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create query builder
    qb := fluentsql.New(db)

    // Select query
    var users []User
    err = qb.Table("users").
        Select("id", "name", "email").
        Where("status", "=", "active").
        WhereIn("role", []interface{}{"admin", "moderator"}).
        OrderBy("created_at", "DESC").
        Limit(10).
        Get(&users)

    // Insert
    result, err := qb.Table("users").Insert(map[string]interface{}{
        "name":  "John Doe",
        "email": "john@example.com",
    })

    // Update
    result, err = qb.Table("users").
        Where("id", "=", 1).
        Update(map[string]interface{}{
            "status": "inactive",
        })

    // Delete
    result, err = qb.Table("users").
        Where("status", "=", "banned").
        Delete()
}

type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}
```

## Documentation

📚 **[Full Documentation](https://pkg.go.dev/github.com/biyonik/go-fluent-sql)**

### Query Building

```go
// Basic WHERE
qb.Where("column", "=", value)
qb.OrWhere("column", "!=", value)

// WHERE IN
qb.WhereIn("status", []interface{}{"active", "pending"})
qb.WhereNotIn("role", []interface{}{"banned"})

// WHERE BETWEEN
qb.WhereBetween("age", 18, 65)
qb.WhereNotBetween("score", 0, 50)

// WHERE NULL
qb.WhereNull("deleted_at")
qb.WhereNotNull("email_verified_at")

// Date queries
qb.WhereDate("created_at", "2024-01-15")
qb.WhereYear("created_at", 2024)
qb.WhereMonth("created_at", 12)

// Ordering
qb.OrderBy("created_at", "DESC")

// Pagination
qb.Limit(10).Offset(20)
```

### Transactions

```go
tx, err := fluentsql.BeginTransaction(db)
if err != nil {
    return err
}

// Use transaction
err = tx.Table("users").Where("id", "=", 1).Update(data)
if err != nil {
    tx.Rollback()
    return err
}

err = tx.Table("logs").Insert(logData)
if err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

### Migrations

```go
migrator := migration.NewMigrator(db, migration.NewMySQLGrammar())

err := migrator.CreateTable("users", func(t *migration.Blueprint) {
    t.ID()
    t.String("name", 255)
    t.String("email", 255).Unique()
    t.String("password", 255)
    t.Timestamps()
    t.SoftDeletes()
})
```

## Benchmarks

```
BenchmarkSelect-8         500000    2340 ns/op    1024 B/op    15 allocs/op
BenchmarkWhere-8          800000    1456 ns/op     512 B/op     8 allocs/op
BenchmarkInsert-8         600000    1892 ns/op     768 B/op    12 allocs/op
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) first.

```bash
# Clone the repo
git clone https://github.com/biyonik/go-fluent-sql.git

# Install dependencies
make deps

# Run tests
make test

# Run linter
make lint
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [go-fluent-validator](https://github.com/biyonik/go-fluent-validator) - Fluent validation library for Go

## Acknowledgments

Inspired by Laravel's Eloquent and Query Builder.
