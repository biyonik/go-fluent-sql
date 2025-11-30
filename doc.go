// Package fluentsql provides a fluent SQL query builder for Go.
//
// go-fluent-sql offers a Laravel-inspired API for building SQL queries
// with built-in protection against SQL injection attacks.
//
// # Quick Start
//
// Connect to a database and start building queries:
//
//	db, err := fluentsql.Connect("user:pass@tcp(localhost:3306)/dbname")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	qb := fluentsql.New(db)
//
// # Select Queries
//
// Build SELECT queries using the fluent API:
//
//	var users []User
//	err := qb.Table("users").
//	    Select("id", "name", "email").
//	    Where("status", "=", "active").
//	    OrderBy("created_at", "DESC").
//	    Limit(10).
//	    Get(&users)
//
// # Where Clauses
//
// Multiple WHERE methods are available:
//
//	qb.Where("age", ">", 18)
//	qb.OrWhere("role", "=", "admin")
//	qb.WhereIn("status", []interface{}{"active", "pending"})
//	qb.WhereBetween("created_at", startDate, endDate)
//	qb.WhereNull("deleted_at")
//
// # Insert, Update, Delete
//
// Execute write operations:
//
//	// Insert
//	result, err := qb.Table("users").Insert(map[string]interface{}{
//	    "name": "John",
//	    "email": "john@example.com",
//	})
//
//	// Update
//	result, err := qb.Table("users").
//	    Where("id", "=", 1).
//	    Update(map[string]interface{}{"status": "inactive"})
//
//	// Delete
//	result, err := qb.Table("users").
//	    Where("status", "=", "banned").
//	    Delete()
//
// # Transactions
//
// Use transactions for atomic operations:
//
//	tx, err := fluentsql.BeginTransaction(db)
//	if err != nil {
//	    return err
//	}
//
//	if err := tx.Table("accounts").Where("id", "=", 1).Update(debit); err != nil {
//	    tx.Rollback()
//	    return err
//	}
//
//	if err := tx.Table("accounts").Where("id", "=", 2).Update(credit); err != nil {
//	    tx.Rollback()
//	    return err
//	}
//
//	return tx.Commit()
//
// # Security
//
// go-fluent-sql protects against SQL injection through:
//   - Prepared statements for all values
//   - Identifier validation (table/column names)
//   - Operator whitelisting
//
// # Thread Safety
//
// QueryBuilder instances are NOT thread-safe. Create a new instance
// for each goroutine or query.
//
// # Supported Databases
//
//   - MySQL / MariaDB
//   - PostgreSQL (coming soon)
//   - SQLite (coming soon)
package fluentsql
