# provider-aws-knowledgebases

A [Crossplane](https://crossplane.io/) provider for managing Amazon Bedrock
Knowledge Bases as Kubernetes resources. Create, update and delete vector
knowledge bases backed by S3 Vectors, OpenSearch Serverless or Pinecone, with
credentials from a Secret or by assuming an IAM role.

## What it manages

One managed resource: `KnowledgeBase`, in API group
`awskb.providerawsknowledgebases.crossplane.io/v1alpha1`. Each one maps to a
single Bedrock knowledge base of type `VECTOR`.

| Vector store | `storageConfiguration.type` | You provide | Example |
|---|---|---|---|
| S3 Vectors | `S3_VECTORS` | Vector bucket ARN, index ARN or name | [basic](examples/knowledgebase/basic-s3-vectors.yaml), [advanced](examples/knowledgebase/advanced-s3-vectors.yaml) |
| OpenSearch Serverless | `OPENSEARCH_SERVERLESS` | Collection ARN, index name, field mapping | [opensearch-serverless.yaml](examples/knowledgebase/opensearch-serverless.yaml) |
| Pinecone | `PINECONE` | Connection string, Secrets Manager ARN for the API key, field mapping | [pinecone.yaml](examples/knowledgebase/pinecone.yaml) |

The knowledge base ID and ARN are written to `status.atProvider` and to the
connection secret as `knowledgeBaseId` and `knowledgeBaseArn`.

## Install

There is no published package yet, so build one and push it to a registry you
control:

```shell
make submodules
make build
make publish XPKG_REG_ORGS=ghcr.io/<you>
```

Then install it into a cluster running Crossplane:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-aws-knowledgebases
spec:
  package: ghcr.io/<you>/provider-aws-knowledgebases:<version>
```

Next, create a `ProviderConfig`. The provider reads an INI-style AWS
credentials file from a Secret, with the profile named `[default]`:

- [`config.yaml`](examples/provider/config.yaml): static access key and
  secret key.
- [`role-assumption-config.yaml`](examples/provider/role-assumption-config.yaml):
  the same, plus `role_arn` and `role_session_name`. The provider assumes that
  role through STS before calling Bedrock.

Each example contains its Secret with placeholder values. Encode your own
credentials file, paste the output over the `credentials:` value, then apply:

```shell
cat > credentials.ini <<EOF
[default]
aws_access_key_id = <access key>
aws_secret_access_key = <secret key>
EOF

base64 < credentials.ini
kubectl apply -f examples/provider/config.yaml
```

Two other examples in `examples/provider/` don't work yet:
[`iam-role-config.yaml`](examples/provider/iam-role-config.yaml) uses a JSON
credential format the provider doesn't parse, and
[`environment-config.yaml`](examples/provider/environment-config.yaml)
expects instance profile or IRSA credentials, which aren't supported. See
[Status](#status).

## Quick start

Create the Bedrock service role and the S3 vector bucket and index first, then
fill in their ARNs. Set `providerConfigRef.name` to the ProviderConfig you
applied (`default` for `config.yaml`). This is
[`examples/knowledgebase/basic-s3-vectors.yaml`](examples/knowledgebase/basic-s3-vectors.yaml):

```yaml
apiVersion: awskb.providerawsknowledgebases.crossplane.io/v1alpha1
kind: KnowledgeBase
metadata:
  name: basic-s3-vectors-kb
  annotations:
    meta.crossplane.io/example: "true"
spec:
  forProvider:
    region: us-east-1
    name: "basic-s3-vectors-knowledge-base"
    description: "A basic knowledge base using S3 vectors for storage"
    roleArn: "arn:aws:iam::123456789012:role/AmazonBedrockExecutionRoleForAgents_TestRole"
    knowledgeBaseConfiguration:
      type: "VECTOR"
      vectorKnowledgeBaseConfiguration:
        embeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"
    storageConfiguration:
      type: "S3_VECTORS"
      s3VectorsConfiguration:
        indexArn: "arn:aws:s3vectors:us-east-1:123456789012:bucket/your-vector-bucket/index/bedrock-knowledge-base-default-index"
        vectorBucketArn: "arn:aws:s3vectors:us-east-1:123456789012:bucket/your-vector-bucket"
    tags:
      Environment: "development"
      Project: "ai-platform"
  providerConfigRef:
    name: role-assumption
```

```shell
kubectl apply -f examples/knowledgebase/basic-s3-vectors.yaml
kubectl get knowledgebase
```

```
NAME                  READY   SYNCED   EXTERNAL-NAME   AGE
basic-s3-vectors-kb   True    True     ABCDE12345      45s
```

`EXTERNAL-NAME` is the Bedrock knowledge base ID. To adopt a knowledge base
that already exists, set the `crossplane.io/external-name` annotation to its ID
before applying.

## Status

Alpha (`v1alpha1`). The `KnowledgeBase` create, read, update and delete calls
work, but:

- Data sources and ingestion jobs aren't implemented. You can create a
  knowledge base but can't load documents into it from Crossplane.
- Only `VECTOR` knowledge bases with the three stores above are supported.
  `KENDRA`, `SQL` and the other storage types (RDS, MongoDB Atlas and so on)
  pass validation, but their settings aren't sent to AWS.
- Drift detection compares only `name`, `description` and `roleArn`. Changes
  to tags, the embedding model or storage settings aren't detected.
- `Ready` becomes `True` as soon as the knowledge base exists, even while
  Bedrock still reports it as `CREATING` or `FAILED`. Check
  `status.atProvider.status`.
- Credentials must be an INI file in a Secret. The `Environment` and
  `InjectedIdentity` credential sources (instance profile, IRSA) aren't
  supported.

## Development

```shell
make submodules   # pull in the shared crossplane/build Makefiles
make reviewable   # code generation, linters and tests
make build        # build the binary, image and package
```

`make dev` creates a kind cluster, installs the CRDs and runs the provider
locally against it. `make dev-clean` deletes the cluster.

See Crossplane's [CONTRIBUTING.md] and the [provider development guide] for
general conventions.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/main/CONTRIBUTING.md
[provider development guide]: https://github.com/crossplane/crossplane/blob/main/contributing/guide-provider-development.md

## License

Apache-2.0. See [LICENSE](LICENSE).
