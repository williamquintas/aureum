# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 0.1.x   | ✅        |
| < 0.1   | ❌        |

## Reporting a Vulnerability

We take the security of Aureum seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **GitHub Security Advisories**: Report a vulnerability (preferred)
2. **Email**: Contact via GitHub for urgent security issues

### What to Include

When reporting a vulnerability, please include:

- **Type of issue** (e.g., XSS, CSRF, SQL injection, etc.)
- **Full paths of source file(s) related to the vulnerability**
- **Location of the affected code** (tag/branch/commit or direct link)
- **Step-by-step instructions to reproduce the issue**
- **Proof-of-concept or exploit code** (if possible)
- **Impact of the vulnerability** (what an attacker might achieve)
- **Suggested fix** (if you have one)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 7 days
- **Updates**: We will provide regular updates on the progress of fixing the vulnerability
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

### Disclosure Policy

- We will work with you to understand and resolve the issue quickly
- We will not disclose the vulnerability publicly until a fix is available
- We will credit you in our security advisories (unless you prefer to remain anonymous)
- We will notify you when the vulnerability is fixed and publicly disclosed

### Security Best Practices

When using this application, please follow these security best practices:

1. **Keep dependencies updated**: Regularly update Go modules to receive security patches
2. **Use environment variables**: Never commit secrets or API keys to the repository
3. **Validate input**: Always validate and sanitize user input
4. **Use HTTPS**: Always use HTTPS in production environments
5. **Regular audits**: Run `go vet` and security scanners regularly

## Security Checklist for Contributors

When contributing code, please ensure:

- [ ] No hardcoded secrets or API keys
- [ ] Input validation and sanitization implemented
- [ ] Proper error handling (no sensitive data in error messages)
- [ ] Dependencies are up to date
- [ ] Security headers configured (if applicable)
- [ ] Authentication and authorization verified
- [ ] SQL injection prevention (parameterized queries)
