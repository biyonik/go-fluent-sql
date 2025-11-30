# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of go-fluent-sql seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: **ahmet.altun60@gmail.com** (replace with actual email)

You should receive a response within 48 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

Please include the following information in your report:

- Type of issue (e.g., SQL injection, buffer overflow, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

### What to Expect

- **Acknowledgment**: We will acknowledge your email within 48 hours.
- **Communication**: We will keep you informed of the progress towards a fix.
- **Credit**: We will credit you in the security advisory (unless you prefer to remain anonymous).
- **Disclosure**: We aim to release a fix within 90 days.

## Security Best Practices for Users

### SQL Injection Prevention

go-fluent-sql is designed to prevent SQL injection attacks through:

1. **Prepared Statements**: All user values are bound via placeholders
2. **Identifier Validation**: Table and column names are validated against a whitelist
3. **Operator Whitelist**: Only approved SQL operators are allowed

However, users should:

```go
// ✅ SAFE: Values are parameterized
qb.Where("status", "=", userInput)

// ❌ UNSAFE: Never concatenate user input
qb.WhereRaw("status = '" + userInput + "'") // DON'T DO THIS
```

### Connection Security

Always use encrypted connections in production:

```go
// Use TLS for MySQL connections
dsn := "user:password@tcp(host:3306)/db?tls=true"
```

### Environment Variables

Never hardcode credentials:

```go
// ✅ GOOD
dsn := os.Getenv("DATABASE_URL")

// ❌ BAD
dsn := "user:password@tcp(localhost)/db"
```

## Known Security Limitations

1. **WhereRaw**: If implemented, allows raw SQL which bypasses protections
2. **Custom Grammar**: Custom grammar implementations must maintain security invariants

## Security Updates

Security updates will be released as:

- Patch versions for minor issues
- Minor versions for significant issues
- Security advisories for critical issues

Subscribe to releases to stay informed about security updates.

## Contact

For any security-related questions, contact: **ahmet.altun60@gmail.com**
