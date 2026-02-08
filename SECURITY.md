# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Kubedial, please report it by emailing the maintainers directly. Do not create a public GitHub issue for security vulnerabilities.

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (optional)

We will acknowledge receipt within 48 hours and provide a more detailed response within 7 days.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Security Best Practices

When deploying Kubedial:

1. **Use TLS**: Always use TLS termination via Ingress for production deployments
2. **Rotate tokens**: Regularly rotate agent and admin bearer tokens
3. **RBAC**: Use minimal RBAC permissions for both kubecommander and kubedialer
4. **Network policies**: Restrict network access to kubecommander
5. **Audit logging**: Enable Kubernetes audit logging to track API access
