# Security Scan Configuration for Focus API

## OWASP Top 10 Checks

### A01: Broken Access Control
- [ ] Test unauthorized access to /api/v1/admin/*
- [ ] Test vertical privilege escalation
- [ ] Test horizontal privilege escalation

### A02: Cryptographic Failures
- [ ] Verify TLS 1.2+ enforced
- [ ] Check JWT signature validation
- [ ] Verify password hashing (bcrypt/argon2)

### A03: Injection
- [ ] SQL injection tests
- [ ] NoSQL injection tests
- [ ] Command injection tests

### A04: Insecure Design
- [x] Review authentication flow
- [ ] Review authorization model
- [x] Review session management

### A05: Security Misconfiguration
- [ ] Check security headers
- [ ] Verify CORS configuration
- [ ] Check error messages (no stack traces)

### A06: Vulnerable Components
- [ ] Run `npm audit` for frontend
- [ ] Run `govulncheck` for backend
- [ ] Check Docker images for CVEs

### A07: Authentication Failures
- [ ] Test brute force protection
- [ ] Test session fixation
- [x] Test JWT expiration

### A08: Software & Data Integrity
- [ ] Verify dependency signatures
- [ ] Check CI/CD pipeline security
- [ ] Review deserialization

### A09: Logging Failures
- [ ] Verify auth events logged
- [ ] Check sensitive data masking
- [ ] Test log injection prevention

### A10: SSRF
- [ ] Test webhook URL validation
- [ ] Test file upload SSRF
- [ ] Test redirect URLs

## Automated Scans

### OWASP ZAP
```bash
# Baseline scan
zap-baseline.py -t http://localhost:8080 -r report.html

# Full scan
zap-full-scan.py -t http://localhost:8080 -r report.html

# API scan
zap-api-scan.py -t swagger.json -f openapi -r report.html
```

### Nuclei
```bash
nuclei -u http://localhost:8080 -o nuclei-report.txt
```

### SQLMap (if applicable)
```bash
sqlmap -u "http://localhost:8080/api/v1/rooms?id=*" --batch
```

## Manual Testing Checklist

### Authentication
- [ ] Login with valid credentials
- [ ] Login with invalid credentials
- [ ] Password reset flow
- [ ] Session timeout
- [ ] Concurrent sessions

### Authorization
- [ ] Access admin endpoints as user
- [ ] Access other user's resources
- [x] Role-based access control

### Input Validation
- [ ] XSS in message content
- [ ] XSS in room names
- [ ] File upload validation

### API Security
- [ ] Rate limiting
- [ ] Input size limits
- [ ] Content-Type validation

## Security Headers

Expected headers:
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
```

## Compliance

- [ ] GDPR compliance check
- [ ] 152-ФЗ compliance check
- [ ] Data retention policy
- [ ] Privacy policy

## Reports

Reports are generated in:
- `tests/security/reports/baseline_report.html`
- `tests/security/reports/full_report.html`
- `tests/security/reports/api_report.html`

## Remediation

For each finding:
1. Assess severity (Critical/High/Medium/Low)
2. Create GitHub issue
3. Fix vulnerability
4. Re-scan to verify fix
5. Document in security log
