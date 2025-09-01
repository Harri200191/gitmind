# Multi-Language Security Analysis

GitMind now supports comprehensive security analysis across multiple programming languages. The security system automatically detects file types and runs appropriate analyzers.

## Supported Languages and Analyzers

### Go
- **gosec**: Go security checker for common security issues
- Detects: hardcoded credentials, unsafe operations, weak crypto

### Python
- **bandit**: Security linter for Python code
- **safety**: Checks Python dependencies for known vulnerabilities
- Detects: SQL injection, code injection, insecure randomness

### JavaScript/TypeScript
- **eslint-security**: ESLint plugin for security issues
- Detects: XSS vulnerabilities, eval usage, object injection

### Ruby
- **brakeman**: Static analysis security scanner for Ruby on Rails
- Detects: SQL injection, XSS, mass assignment, CSRF

### Java
- **spotbugs**: Static analysis tool for Java bytecode
- Detects: security bugs, performance issues, correctness problems

### PHP
- **psalm**: Static analysis tool for PHP
- **phpstan**: PHP static analysis tool
- Detects: type errors, security vulnerabilities, code quality issues

### C/C++
- **cppcheck**: Static analysis tool for C/C++ code
- **flawfinder**: Scans C/C++ source code for security flaws
- Detects: buffer overflows, format string vulnerabilities, race conditions

### Rust
- **cargo-audit**: Audit Cargo.lock files for crates with security vulnerabilities
- **clippy**: Rust linter for catching common mistakes
- Detects: dependency vulnerabilities, unsafe code patterns

### Multi-Language
- **semgrep**: Static analysis tool supporting many languages
- **securecodewarrior**: Comprehensive security scanner (enterprise)
- Detects: OWASP Top 10, custom security rules

## Configuration

Enable security analysis in your `.gitmind.yaml`:

```yaml
security:
  enabled: true
  analyzers: [
    "gosec",           # Go security
    "bandit",          # Python security
    "safety",          # Python dependencies
    "eslint-security", # JavaScript/TypeScript
    "brakeman",        # Ruby on Rails
    "spotbugs",        # Java
    "psalm",           # PHP
    "phpstan",         # PHP
    "cppcheck",        # C/C++
    "flawfinder",      # C/C++ security
    "cargo-audit",     # Rust dependencies
    "clippy",          # Rust linting
    "semgrep"          # Multi-language
  ]
  block_on_high: false    # Set to true to block commits with high-severity issues
  include_in_msg: true    # Include security notes in commit messages
```

## Usage

### Basic Security Check
```bash
gitmind security-check
```

### Verbose Output
```bash
gitmind security-check --verbose
```

### Block on High Severity
```bash
gitmind security-check --block
```

## Pattern-Based Analysis

GitMind also includes built-in pattern-based analysis for common security issues across all languages:

- **Hardcoded credentials**: Passwords, API keys, tokens
- **Code injection**: eval(), exec(), dynamic code execution
- **SQL injection**: String concatenation in SQL queries
- **Command injection**: Unsafe system calls
- **Insecure protocols**: HTTP instead of HTTPS
- **Language-specific patterns**: Unsafe functions for each language

## Installation Requirements

For full functionality, install the relevant analyzers:

```bash
# Go
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Python
pip install bandit safety

# JavaScript/TypeScript
npm install -g eslint eslint-plugin-security

# Ruby
gem install brakeman

# Java (requires separate installation)
# Download from https://spotbugs.github.io/

# PHP
composer global require vimeo/psalm phpstan/phpstan

# C/C++
# Install via package manager: apt-get install cppcheck flawfinder

# Rust
cargo install cargo-audit

# Multi-language
pip install semgrep
```

## Integration with Commit Hooks

Security analysis automatically runs during commit message generation when enabled. High-severity issues can optionally block commits:

```bash
# Stage changes
git add .

# Commit - security analysis runs automatically
git commit
```

## Output Example

```
🔒 Security Analysis Results:
  Total findings: 3
  High severity: 1
  Medium severity: 1
  Low severity: 1

Detailed Findings:
  🔴 [high] main.py:15 - Hardcoded password detected
    💡 Use environment variables or secure configuration
  🟡 [medium] app.js:42 - Potential XSS via innerHTML
    💡 Use textContent or sanitize HTML
  🟢 [low] server.go:78 - Insecure HTTP protocol detected
    💡 Use HTTPS instead

General Suggestions:
  • Validate and sanitize all inputs
  • Use environment variables for secrets
  • Follow secure coding practices
```

This multi-language approach ensures comprehensive security coverage regardless of your technology stack!