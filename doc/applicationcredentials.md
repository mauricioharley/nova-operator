# Application Credentials Support in Nova Operator

## Overview

Application Credentials provide a secure way for applications to authenticate with OpenStack services without exposing user passwords. The Nova operator implements support for managing Application Credentials through integration with the Keystone operator.

## Features

The Nova operator's Application Credentials implementation provides:

- **Automated Credential Management**: Creates and manages Application Credentials for Nova service users
- **Rotation Support**: Automatically rotates credentials based on configured expiration and grace periods
- **Secure Storage**: Stores credentials securely in Kubernetes Secrets
- **Integration with Keystone**: Seamless integration with KeystoneAPI resources

## Architecture

The Application Credentials feature is implemented through:

1. **ApplicationCredentialReconciler**: A controller that watches ApplicationCredential resources
2. **Secret Management**: Automatic creation and management of Kubernetes Secrets containing credentials
3. **Keystone Integration**: Direct integration with Keystone API for credential lifecycle management

## Prerequisites

Before using Application Credentials with the Nova operator, ensure:

1. **Keystone Operator**: The keystone-operator must be deployed and operational
2. **KeystoneAPI Resource**: A KeystoneAPI resource must be Ready
3. **Service User**: The target service user (e.g., 'nova') must exist in Keystone
4. **Service Password**: A Secret containing the service user's password must exist

## Usage

### Basic ApplicationCredential Resource

> **Note**: ApplicationCredential types are currently being added to the keystone-operator API.
> The following examples will be available once the types are implemented.

```yaml
apiVersion: keystone.openstack.org/v1beta1
kind: ApplicationCredential
metadata:
  name: nova-applicationcredential
  namespace: openstack
spec:
  userName: nova
  secret: nova-service-password
  expirationDays: 30
  gracePeriodDays: 7
  keystoneAPIName: keystone
```

### ApplicationCredential with Custom Configuration

```yaml
apiVersion: keystone.openstack.org/v1beta1
kind: ApplicationCredential
metadata:
  name: nova-compute-credential
  namespace: openstack
spec:
  userName: nova
  secret: nova-service-password
  expirationDays: 90    # 90 days until expiration
  gracePeriodDays: 14   # Start rotation 14 days before expiration
  keystoneAPIName: keystone
```

## Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `userName` | string | Yes | Name of the service user to create credentials for |
| `secret` | string | Yes | Name of the Secret containing the user's password |
| `expirationDays` | int | No | Number of days until credential expires (default: no expiration) |
| `gracePeriodDays` | int | No | Days before expiration to start rotation |
| `keystoneAPIName` | string | No | Name of the KeystoneAPI resource to use |

## Credential Rotation

Application Credentials support automatic rotation based on the configured parameters:

- **Rotation Trigger**: Credentials are rotated when the current time is within the grace period before expiration
- **Rotation Schedule**: The controller reconciles at least once daily to check for rotation needs
- **Old Credentials**: Previous credentials are not explicitly revoked as they naturally expire

### Rotation Example

For a credential with:
- `expirationDays: 30`
- `gracePeriodDays: 7`

The rotation process:
1. Credential created on Day 0
2. Rotation window starts on Day 23 (30 - 7)
3. Next reconciliation after Day 23 triggers rotation
4. New credential created with 30-day expiration
5. Old credential expires naturally on Day 30

## Secret Management

The Nova operator automatically creates and manages Kubernetes Secrets containing Application Credentials:

### Secret Naming

Application Credential secrets follow the pattern: `{instance-name}-application-credential`

### Secret Structure

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nova-applicationcredential-application-credential
  namespace: openstack
  labels:
    app.kubernetes.io/name: nova
    managed-by: nova-operator
  annotations:
    created-at: "2024-01-15T10:30:00Z"
type: Opaque
data:
  id: <base64-encoded-credential-id>
  secret: <base64-encoded-credential-secret>
```

## Integration with Nova Services

Once Application Credentials are available, Nova services can be configured to use them for authentication:

1. **Service Configuration**: Update Nova service configurations to use application credential authentication
2. **Secret References**: Point Nova services to the application credential secrets
3. **Authentication**: Services authenticate using the application credential ID and secret

## Status and Conditions

ApplicationCredential resources provide status information through conditions:

- **Ready**: Overall readiness of the Application Credential
- **InputReady**: Input secrets and dependencies are available
- **KeystoneAPIReady**: KeystoneAPI resource is ready and accessible
- **ApplicationCredentialReady**: Application Credential is created and functional

## Troubleshooting

### Common Issues

1. **KeystoneAPI Not Ready**
   - Ensure KeystoneAPI resource exists and is in Ready state
   - Check KeystoneAPI connectivity and configuration

2. **Service Password Secret Missing**
   - Verify the specified secret exists and contains 'ServicePassword' key
   - Check secret permissions and accessibility

3. **Authentication Failures**
   - Verify service user exists in Keystone
   - Check service user password is correct
   - Ensure proper domain and project configuration

### Debugging

Check ApplicationCredential resource status:
```bash
kubectl get applicationcredential nova-applicationcredential -o yaml
```

Examine controller logs:
```bash
kubectl logs -l app.kubernetes.io/name=nova-operator -f
```

## Security Considerations

1. **Secret Protection**: Application Credential secrets contain sensitive authentication data
2. **RBAC**: Ensure appropriate RBAC permissions for accessing secrets
3. **Rotation**: Regular rotation reduces exposure risk
4. **Monitoring**: Monitor credential usage and expiration

## RBAC Requirements

The Nova operator requires the following RBAC permissions for Application Credentials:

```yaml
- apiGroups: ["keystone.openstack.org"]
  resources: ["applicationcredentials"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["keystone.openstack.org"]
  resources: ["applicationcredentials/status", "applicationcredentials/finalizers"]
  verbs: ["get", "update", "patch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Migration from Password Authentication

When migrating from password-based authentication to Application Credentials:

1. **Create Application Credentials**: Deploy ApplicationCredential resources
2. **Update Service Configuration**: Modify Nova services to use application credentials
3. **Verify Functionality**: Test that services work with new authentication
4. **Monitor**: Ensure smooth operation before removing password authentication

## Testing

The Nova operator includes comprehensive tests for Application Credentials:

### KUTTL Tests

Location: `test/kuttl/tests/applicationcredential/`

These tests verify:
- ApplicationCredential resource creation
- Secret generation and management
- Integration with KeystoneAPI

### Functional Tests

Location: `test/functional/applicationcredential_test.go`

These tests cover:
- Application Credential lifecycle
- Rotation functionality
- Error handling and edge cases
- Integration with Nova services

## Future Enhancements

Planned improvements include:

1. **Enhanced Rotation Policies**: More flexible rotation schedules
2. **Role-based Credentials**: Support for application credentials with specific roles
3. **Multi-project Support**: Credentials spanning multiple projects
4. **Monitoring Integration**: Metrics and alerts for credential lifecycle

## References

- [OpenStack Application Credentials Documentation](https://docs.openstack.org/keystone/latest/user/application_credentials.html)
- [Keystone API Reference](https://docs.openstack.org/api-ref/identity/v3/#application-credentials)
- [Keystone Operator PR #567](https://github.com/openstack-k8s-operators/keystone-operator/pull/567)
