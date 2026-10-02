# ProviderConfig Examples

This directory contains example ProviderConfig manifests for configuring the AWS Knowledge Bases provider with different authentication methods.

## Prerequisites

Before using these examples, ensure you have:

1. A Crossplane cluster with the AWS Knowledge Bases provider installed
2. Appropriate AWS IAM permissions for the credentials you're using
3. The `crossplane-system` namespace created (or modify the namespace in the examples)

## Authentication Methods

### 1. Secret-based Authentication (`config.yaml`)

The most common method using AWS access keys stored in a Kubernetes Secret.

**Required AWS Permissions:**
- `bedrock:*` (for Bedrock Agent operations)
- `s3:*` (for S3 vector storage, if used)
- `aoss:*` (for OpenSearch Serverless, if used)
- `secretsmanager:GetSecretValue` (for Pinecone credentials, if used)

**Setup:**
1. Create AWS access keys with the required permissions
2. Base64 encode your credentials JSON:
   ```bash
   echo '{"accessKeyId":"YOUR_ACCESS_KEY","secretAccessKey":"YOUR_SECRET_KEY"}' | base64
   ```
3. Replace the `credentials` field in the Secret with your encoded credentials
4. Apply the manifest:
   ```bash
   kubectl apply -f config.yaml
   ```

### 2. IAM Role Authentication (`iam-role-config.yaml`)

Uses IAM role assumption for enhanced security. This method allows you to use temporary credentials and assume a role with specific permissions.

**Required Setup:**
1. Create an IAM role with the necessary permissions
2. Configure trust relationships to allow your base credentials to assume the role
3. Base64 encode your credentials JSON with role information:
   ```bash
   echo '{"accessKeyId":"YOUR_ACCESS_KEY","secretAccessKey":"YOUR_SECRET_KEY","roleArn":"arn:aws:iam::123456789012:role/YourRole","sessionName":"crossplane-session"}' | base64
   ```

**Benefits:**
- Enhanced security through role assumption
- Temporary credentials with limited lifetime
- Easier permission management through IAM roles

### 3. Environment-based Authentication (`environment-config.yaml`)

Uses AWS credentials from the environment where the provider is running. This is useful when running on EC2 instances with IAM instance profiles or in EKS with IAM roles for service accounts (IRSA).

**Setup Options:**

#### Option A: EC2 Instance Profile
1. Create an IAM role with the required permissions
2. Attach the role to your EC2 instances running Crossplane
3. Apply the environment config:
   ```bash
   kubectl apply -f environment-config.yaml
   ```

#### Option B: EKS with IAM Roles for Service Accounts (IRSA)
1. Create an IAM role with the required permissions
2. Configure IRSA for the provider's service account
3. Apply the environment config

**Benefits:**
- No need to manage static credentials
- Automatic credential rotation
- Better security posture

## Usage

After applying a ProviderConfig, reference it in your KnowledgeBase resources:

```yaml
apiVersion: awskb.template.crossplane.io/v1alpha1
kind: KnowledgeBase
metadata:
  name: my-knowledge-base
spec:
  providerConfigRef:
    name: default  # or iam-role, environment, etc.
  forProvider:
    # ... your KnowledgeBase configuration
```

## Security Best Practices

1. **Use IAM Roles**: Prefer IAM role assumption over static access keys
2. **Least Privilege**: Grant only the minimum required permissions
3. **Rotate Credentials**: Regularly rotate access keys and update secrets
4. **Monitor Usage**: Use AWS CloudTrail to monitor API usage
5. **Environment Variables**: Use environment-based auth when possible (EC2 instance profiles, IRSA)

## Required IAM Permissions

Here's a minimal IAM policy for the AWS Knowledge Bases provider:

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
                "bedrock:ListKnowledgeBases"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::your-vector-bucket",
                "arn:aws:s3:::your-vector-bucket/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "aoss:CreateCollection",
                "aoss:DeleteCollection",
                "aoss:UpdateCollection",
                "aoss:BatchGetCollection"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue"
            ],
            "Resource": "arn:aws:secretsmanager:*:*:secret:*pinecone*"
        }
    ]
}
```

## Troubleshooting

1. **Authentication Errors**: Verify that your credentials are correctly base64 encoded and have the required permissions
2. **Permission Denied**: Check that your IAM user/role has the necessary permissions for Bedrock and other AWS services
3. **Region Issues**: Ensure that Bedrock is available in your target region
4. **Secret Not Found**: Verify that the secret exists in the correct namespace and has the right key name

For more information, refer to the [AWS Bedrock documentation](https://docs.aws.amazon.com/bedrock/latest/userguide/security-iam.html).
