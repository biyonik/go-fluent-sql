# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project setup
- Query Builder with fluent API
- MySQL Grammar support
- SQL injection protection
- Prepared statements
- WHERE clause methods (Where, OrWhere, WhereIn, WhereBetween, WhereNull, etc.)
- ORDER BY with direction validation
- LIMIT and OFFSET support
- Transaction support
- Migration system with Blueprint
- Redis connection pool
- Struct scanning with reflection caching

### Security
- Identifier validation with regex whitelist
- Operator whitelist validation
- Prepared statement parameter binding

## [0.1.0] - YYYY-MM-DD

### Added
- Initial alpha release

---

## Version History

### Versioning Scheme

- **MAJOR**: Incompatible API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Pre-release Tags

- `alpha`: Early development, unstable
- `beta`: Feature complete, testing
- `rc`: Release candidate, final testing

[Unreleased]: https://github.com/biyonik/go-fluent-sql/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/biyonik/go-fluent-sql/releases/tag/v0.1.0
