# ProviderConfig examples

Each file creates a `ProviderConfig` named `default`, which is what the
[knowledge base examples](../knowledgebase/) reference. Pick one.

| File | Credentials come from | Use when |
|---|---|---|
| [`config.yaml`](config.yaml) | Access key in a Secret | Local clusters, quick tests |
| [`role-assumption-config.yaml`](role-assumption-config.yaml) | Access key in a Secret, then STS `AssumeRole` | The key's user should only be able to assume a scoped role |
| [`injected-identity-config.yaml`](injected-identity-config.yaml) | The provider pod's own identity | EKS with IRSA or Pod Identity, or EC2 instance profiles |

## Secret-based credentials

`config.yaml` and `role-assumption-config.yaml` read an AWS credentials file in
INI format from the `credentials` key of a Secret. The provider reads the
`[default]` profile:

```ini
[default]
aws_access_key_id = <access key>
aws_secret_access_key = <secret key>
# optional
aws_session_token = <session token>
# optional: assume this role before calling Bedrock
role_arn = arn:aws:iam::123456789012:role/CrossplaneProviderAWSKnowledgeBasesRole
role_session_name = crossplane-provider-session
```

Both examples include their Secret with placeholder values. Replace the
`credentials:` value with your own file, base64-encoded, then apply:

```shell
base64 < credentials.ini
kubectl apply -f config.yaml
```

Or create the Secret yourself and apply only the `ProviderConfig` part of the
file:

```shell
kubectl create secret generic aws-creds -n crossplane-system \
  --from-file=credentials=credentials.ini
```

With `role_arn` set, the access key only needs `sts:AssumeRole` on that role,
and the role needs the permissions below.

## Injected identity

`injected-identity-config.yaml` uses `source: InjectedIdentity`. The provider
loads credentials through the default AWS SDK chain, so whatever identity the
provider pod has is used. No Secret is involved.

For IRSA, annotate the provider's service account with the role through a
`DeploymentRuntimeConfig`:

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: irsa
spec:
  serviceAccountTemplate:
    metadata:
      annotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/CrossplaneProviderAWSKnowledgeBasesRole
```

Then set `spec.runtimeConfigRef.name: irsa` on the `Provider`. The role's
trust policy must allow the cluster's OIDC provider for that service account.

## IAM permissions

The identity the provider runs as calls the Bedrock knowledge base APIs and
passes the knowledge base's service role (`spec.forProvider.roleArn`) to
Bedrock. It doesn't touch S3, OpenSearch or Secrets Manager itself; the
service role does.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:CreateKnowledgeBase",
        "bedrock:GetKnowledgeBase",
        "bedrock:UpdateKnowledgeBase",
        "bedrock:DeleteKnowledgeBase",
        "bedrock:TagResource"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": "arn:aws:iam::123456789012:role/AmazonBedrockExecutionRoleForKnowledgeBase*",
      "Condition": {
        "StringEquals": { "iam:PassedToService": "bedrock.amazonaws.com" }
      }
    }
  ]
}
```

The service role itself needs access to the embedding model and the vector
store. See [Create a service role for Amazon Bedrock Knowledge Bases][kb-role].

## Troubleshooting

- **`missing required AWS credentials`**: the Secret isn't in INI format, or
  has no `[default]` profile with an access key and secret key.
- **`AccessDenied` on `iam:PassRole`**: the provider's identity can't pass the
  knowledge base's `roleArn` to Bedrock.
- **`cannot get credentials`**: the Secret named in `secretRef` doesn't exist,
  or the key isn't `credentials`.

[kb-role]: https://docs.aws.amazon.com/bedrock/latest/userguide/kb-permissions.html
