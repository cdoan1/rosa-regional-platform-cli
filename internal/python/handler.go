package python

// const DefaultHandler = `def lambda_handler(event, context):
//     return {
//         'statusCode': 200,
//         'body': 'hello world'
//     }
// `

const DefaultHandler = `import json
import os
from datetime import datetime, timezone

def lambda_handler(event, context):
    # Get current time
    now = datetime.now(timezone.utc)

    # Get creation time from environment variable
    created_at_str = os.environ.get('CREATED_AT', now.isoformat())
    created_at = datetime.fromisoformat(created_at_str.replace('Z', '+00:00'))

    # Calculate delta
    delta = now - created_at

    # Format ago string
    total_seconds = int(delta.total_seconds())
    if total_seconds < 60:
        ago = f"{total_seconds} seconds ago"
    elif total_seconds < 3600:
        minutes = total_seconds // 60
        ago = f"{minutes} minute{'s' if minutes != 1 else ''} ago"
    elif total_seconds < 86400:
        hours = total_seconds // 3600
        ago = f"{hours} hour{'s' if hours != 1 else ''} ago"
    else:
        days = total_seconds // 86400
        ago = f"{days} day{'s' if days != 1 else ''} ago"

    return {
        'statusCode': 200,
        'body': json.dumps({
            'current_time': now.isoformat(),
            'created_at': created_at.isoformat(),
            'message': 'hello world',
            'ago': ago
        })
    }
`

const OIDCHandler = `import json
import boto3
import hashlib
import os

def lambda_handler(event, context):
    """
    Creates an S3-backed OIDC issuer with IAM OIDC provider.

    Expected event:
    {
        "bucket_name": "oidc-issuer-{unique-id}",
        "region": "us-east-1"  (optional, defaults to AWS_REGION)
    }

    RSA public key components are read from environment variables:
    - JWK_N: base64url-encoded modulus
    - JWK_E: base64url-encoded exponent
    - JWK_KID: key ID

    This function:
    1. Creates S3 bucket with public read access
    2. Uploads OIDC discovery documents (.well-known/openid-configuration, keys.json)
    3. Creates IAM OIDC provider pointing to S3 bucket URL

    Returns:
    {
        "bucket_name": "oidc-issuer-xxx",
        "issuer_url": "https://...",
        "provider_arn": "arn:aws:iam::...",
        "discovery_url": "https://.../well-known/openid-configuration",
        "jwks_url": "https://.../keys.json"
    }
    """
    try:
        bucket_name = event.get('bucket_name')
        region = event.get('region', os.environ.get('AWS_REGION', 'us-east-1'))

        # Validate required fields
        if not bucket_name:
            return {
                'statusCode': 400,
                'body': json.dumps({'error': 'bucket_name is required in event payload'})
            }

        # Read JWK components from environment variables
        n_b64 = os.environ.get('JWK_N')
        e_b64 = os.environ.get('JWK_E')
        key_id = os.environ.get('JWK_KID')

        if not all([n_b64, e_b64, key_id]):
            return {
                'statusCode': 500,
                'body': json.dumps({'error': 'JWK environment variables not configured. Please recreate the Lambda function.'})
            }

        s3 = boto3.client('s3', region_name=region)
        iam = boto3.client('iam')

        print(f"Using embedded RSA public key (kid: {key_id})")

        # Step 2: Create S3 bucket
        print(f"Creating S3 bucket: {bucket_name}")
        try:
            if region == 'us-east-1':
                s3.create_bucket(Bucket=bucket_name)
            else:
                s3.create_bucket(
                    Bucket=bucket_name,
                    CreateBucketConfiguration={'LocationConstraint': region}
                )
        except s3.exceptions.BucketAlreadyExists:
            print(f"Bucket {bucket_name} already exists, continuing...")
        except s3.exceptions.BucketAlreadyOwnedByYou:
            print(f"Bucket {bucket_name} already owned by you, continuing...")

        # Step 3: Configure public access
        print("Configuring bucket for public access...")
        s3.put_public_access_block(
            Bucket=bucket_name,
            PublicAccessBlockConfiguration={
                'BlockPublicAcls': False,
                'IgnorePublicAcls': False,
                'BlockPublicPolicy': False,
                'RestrictPublicBuckets': False
            }
        )

        bucket_policy = {
            "Version": "2012-10-17",
            "Statement": [{
                "Sid": "PublicReadGetObject",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:GetObject",
                "Resource": f"arn:aws:s3:::{bucket_name}/*"
            }]
        }
        s3.put_bucket_policy(Bucket=bucket_name, Policy=json.dumps(bucket_policy))

        # Step 4: Generate and upload OIDC documents
        issuer_url = f"https://{bucket_name}.s3.{region}.amazonaws.com"

        oidc_config = {
            "issuer": issuer_url,
            "jwks_uri": f"{issuer_url}/keys.json",
            "response_types_supported": ["id_token"],
            "subject_types_supported": ["public"],
            "id_token_signing_alg_values_supported": ["RS256"]
        }

        jwks = {
            "keys": [{
                "kty": "RSA",
                "use": "sig",
                "kid": key_id,
                "n": n_b64,
                "e": e_b64,
                "alg": "RS256"
            }]
        }

        s3.put_object(
            Bucket=bucket_name,
            Key='.well-known/openid-configuration',
            Body=json.dumps(oidc_config),
            ContentType='application/json'
        )

        s3.put_object(
            Bucket=bucket_name,
            Key='keys.json',
            Body=json.dumps(jwks),
            ContentType='application/json'
        )

        # Step 5: Create IAM OIDC provider
        print("Creating IAM OIDC provider...")
        thumbprint = hashlib.sha1(issuer_url.encode()).hexdigest()

        try:
            provider_response = iam.create_open_id_connect_provider(
                Url=issuer_url,
                ClientIDList=['sts.amazonaws.com'],
                ThumbprintList=[thumbprint]
            )
            provider_arn = provider_response['OpenIDConnectProviderArn']
        except iam.exceptions.EntityAlreadyExistsException:
            providers = iam.list_open_id_connect_providers()
            provider_arn = None
            for provider in providers['OpenIDConnectProviderList']:
                details = iam.get_open_id_connect_provider(
                    OpenIDConnectProviderArn=provider['Arn']
                )
                if details['Url'] == issuer_url:
                    provider_arn = provider['Arn']
                    break
            if not provider_arn:
                raise Exception("OIDC provider exists but couldn't find ARN")

        return {
            'statusCode': 200,
            'body': json.dumps({
                'bucket_name': bucket_name,
                'issuer_url': issuer_url,
                'provider_arn': provider_arn,
                'discovery_url': f"{issuer_url}/.well-known/openid-configuration",
                'jwks_url': f"{issuer_url}/keys.json",
                'message': 'OIDC issuer created successfully'
            })
        }
    except Exception as e:
        print(f"Error: {str(e)}")
        return {
            'statusCode': 500,
            'body': json.dumps({'error': str(e)})
        }
`

const OIDCDeleteHandler = `import json
import boto3
import os

def lambda_handler(event, context):
    """
    Deletes an S3-backed OIDC issuer and its IAM OIDC provider.

    Expected event:
    {
        "bucket_name": "oidc-issuer-{unique-id}",
        "region": "us-east-1"  (optional, defaults to AWS_REGION)
    }

    This function:
    1. Deletes all objects in the S3 bucket
    2. Deletes the S3 bucket
    3. Deletes the IAM OIDC provider

    Returns:
    {
        "bucket_name": "oidc-issuer-xxx",
        "message": "OIDC issuer deleted successfully"
    }
    """
    try:
        bucket_name = event.get('bucket_name')
        region = event.get('region', os.environ.get('AWS_REGION', 'us-east-1'))

        # Validate required fields
        if not bucket_name:
            return {
                'statusCode': 400,
                'body': json.dumps({'error': 'bucket_name is required in event payload'})
            }

        s3 = boto3.client('s3', region_name=region)
        iam = boto3.client('iam')

        print(f"Deleting OIDC issuer: {bucket_name}")

        # Step 1: Delete all objects in the bucket (handle pagination)
        print(f"Deleting objects from bucket: {bucket_name}")
        try:
            # Use paginator to handle buckets with > 1000 objects
            paginator = s3.get_paginator('list_objects_v2')
            pages = paginator.paginate(Bucket=bucket_name)

            objects_deleted = 0
            for page in pages:
                if 'Contents' in page:
                    objects_to_delete = [{'Key': obj['Key']} for obj in page['Contents']]

                    if objects_to_delete:
                        delete_response = s3.delete_objects(
                            Bucket=bucket_name,
                            Delete={'Objects': objects_to_delete}
                        )
                        deleted_count = len(delete_response.get('Deleted', []))
                        objects_deleted += deleted_count
                        print(f"Deleted {deleted_count} objects from this page")

            if objects_deleted > 0:
                print(f"Total objects deleted: {objects_deleted}")
            else:
                print("No objects found in bucket")
        except s3.exceptions.NoSuchBucket:
            print(f"Bucket {bucket_name} not found, skipping S3 deletion")
        except Exception as e:
            print(f"Warning: Failed to delete objects: {str(e)}")
            import traceback
            traceback.print_exc()

        # Step 2: Delete the bucket
        print(f"Deleting bucket: {bucket_name}")
        try:
            s3.delete_bucket(Bucket=bucket_name)
            print(f"Deleted bucket: {bucket_name}")
        except s3.exceptions.NoSuchBucket:
            print(f"Bucket {bucket_name} already deleted")
        except Exception as e:
            print(f"Warning: Failed to delete bucket: {str(e)}")

        # Step 3: Delete the IAM OIDC provider
        issuer_url_with_https = f"https://{bucket_name}.s3.{region}.amazonaws.com"
        issuer_url_without_https = f"{bucket_name}.s3.{region}.amazonaws.com"
        print(f"Deleting IAM OIDC provider for: {issuer_url_with_https}")

        provider_arn = None
        try:
            # Find the OIDC provider ARN
            providers = iam.list_open_id_connect_providers()
            for provider in providers['OpenIDConnectProviderList']:
                details = iam.get_open_id_connect_provider(
                    OpenIDConnectProviderArn=provider['Arn']
                )
                # Check both with and without https:// prefix
                if details['Url'] in [issuer_url_with_https, issuer_url_without_https]:
                    provider_arn = provider['Arn']
                    break

            if provider_arn:
                iam.delete_open_id_connect_provider(
                    OpenIDConnectProviderArn=provider_arn
                )
                print(f"Deleted OIDC provider: {provider_arn}")
            else:
                print(f"OIDC provider not found for {issuer_url_with_https}")
        except Exception as e:
            print(f"Warning: Failed to delete OIDC provider: {str(e)}")

        return {
            'statusCode': 200,
            'body': json.dumps({
                'bucket_name': bucket_name,
                'issuer_url': issuer_url_with_https,
                'provider_arn': provider_arn if provider_arn else 'not found',
                'message': 'OIDC issuer deleted successfully'
            })
        }
    except Exception as e:
        print(f"Error: {str(e)}")
        return {
            'statusCode': 500,
            'body': json.dumps({'error': str(e)})
        }
`
