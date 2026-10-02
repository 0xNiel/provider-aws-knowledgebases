# KnowledgeBase Examples

This directory contains example manifests for creating AWS Bedrock Knowledge Bases using the Crossplane provider.

## Prerequisites

Before using these examples, ensure you have:

1. A Crossplane cluster with the AWS Knowledge Bases provider installed
2. Appropriate AWS IAM roles and permissions configured
3. Required AWS resources (S3 buckets, OpenSearch collections, etc.) created
4. A ProviderConfig configured with AWS credentials

## Examples

### 1. Basic S3 Vectors (`basic-s3-vectors.yaml`)

A simple knowledge base using S3 vectors for storage. This is the most straightforward configuration and requires minimal AWS resources.

**Required AWS Resources:**
- IAM role with Bedrock and S3 permissions
- S3 bucket for vector storage

### 2. OpenSearch Serverless (`opensearch-serverless.yaml`)

A knowledge base using Amazon OpenSearch Serverless for vector storage. This provides more advanced search capabilities.

**Required AWS Resources:**
- IAM role with Bedrock and OpenSearch permissions
- OpenSearch Serverless collection
- Appropriate network and security policies

### 3. Pinecone (`pinecone.yaml`)

A knowledge base using Pinecone as the vector database. This is useful when you want to use a managed vector database service.

**Required AWS Resources:**
- IAM role with Bedrock permissions
- AWS Secrets Manager secret containing Pinecone API key
- Pinecone account and index

### 4. Advanced S3 Vectors (`advanced-s3-vectors.yaml`)

An advanced S3 vectors configuration with custom embedding model settings and additional metadata.

**Required AWS Resources:**
- IAM role with Bedrock and S3 permissions
- S3 bucket for vector storage with custom index configuration

## Usage

1. **Update the ARNs and identifiers** in the examples to match your AWS resources:
   - `roleArn`: Your IAM role ARN
   - `vectorBucketArn`: Your S3 bucket ARN (for S3 examples)
   - `collectionArn`: Your OpenSearch collection ARN (for OpenSearch example)
   - `credentialsSecretArn`: Your Secrets Manager secret ARN (for Pinecone example)

2. **Apply the manifest**:
   ```bash
   kubectl apply -f basic-s3-vectors.yaml
   ```

3. **Check the status**:
   ```bash
   kubectl get knowledgebase
   kubectl describe knowledgebase basic-s3-vectors-kb
   ```

## Configuration Options

### Embedding Models

The examples use different embedding models:
- `amazon.titan-embed-text-v1`: Basic Titan embedding model
- `amazon.titan-embed-text-v2:0`: Advanced Titan embedding model
- `cohere.embed-english-v3`: Cohere embedding model

### Vector Storage Types

- **S3_VECTORS**: Store vectors in Amazon S3 (simplest option)
- **OPENSEARCH_SERVERLESS**: Use OpenSearch Serverless for advanced search
- **PINECONE**: Use Pinecone managed vector database

### Tags

All examples include tags for:
- Environment classification
- Project organization
- Cost tracking
- Ownership identification

## Troubleshooting

1. **Permission Issues**: Ensure your IAM role has the necessary permissions for Bedrock, and the specific vector storage service you're using.

2. **Resource Not Found**: Verify that all referenced AWS resources (buckets, collections, secrets) exist and are in the correct region.

3. **Network Issues**: For OpenSearch Serverless, ensure network policies allow access from Bedrock.

4. **Embedding Model Availability**: Verify that the embedding model is available in your chosen region.

## Next Steps

After creating a knowledge base, you can:
1. Create data sources to populate the knowledge base
2. Associate the knowledge base with Bedrock agents
3. Use the knowledge base for retrieval-augmented generation (RAG)

For more information, refer to the [AWS Bedrock documentation](https://docs.aws.amazon.com/bedrock/latest/userguide/knowledge-base.html).
