// Code generated by smithy-go-codegen DO NOT EDIT.

package kms

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Creates a friendly name for a KMS key.
//
// Adding, deleting, or updating an alias can allow or deny permission to the KMS
// key. For details, see [ABAC for KMS]in the Key Management Service Developer Guide.
//
// You can use an alias to identify a KMS key in the KMS console, in the DescribeKey
// operation and in [cryptographic operations], such as Encrypt and GenerateDataKey. You can also change the KMS key that's
// associated with the alias (UpdateAlias ) or delete the alias (DeleteAlias ) at any time. These
// operations don't affect the underlying KMS key.
//
// You can associate the alias with any customer managed key in the same Amazon
// Web Services Region. Each alias is associated with only one KMS key at a time,
// but a KMS key can have multiple aliases. A valid KMS key is required. You can't
// create an alias without a KMS key.
//
// The alias must be unique in the account and Region, but you can have aliases
// with the same name in different Regions. For detailed information about aliases,
// see [Using aliases]in the Key Management Service Developer Guide.
//
// This operation does not return a response. To get the alias that you created,
// use the ListAliasesoperation.
//
// The KMS key that you use for this operation must be in a compatible key state.
// For details, see [Key states of KMS keys]in the Key Management Service Developer Guide.
//
// Cross-account use: No. You cannot perform this operation on an alias in a
// different Amazon Web Services account.
//
// # Required permissions
//
// [kms:CreateAlias]
//   - on the alias (IAM policy).
//
// [kms:CreateAlias]
//   - on the KMS key (key policy).
//
// For details, see [Controlling access to aliases] in the Key Management Service Developer Guide.
//
// Related operations:
//
// # DeleteAlias
//
// # ListAliases
//
// # UpdateAlias
//
// Eventual consistency: The KMS API follows an eventual consistency model. For
// more information, see [KMS eventual consistency].
//
// [Key states of KMS keys]: https://docs.aws.amazon.com/kms/latest/developerguide/key-state.html
// [cryptographic operations]: https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#cryptographic-operations
// [Using aliases]: https://docs.aws.amazon.com/kms/latest/developerguide/kms-alias.html
// [kms:CreateAlias]: https://docs.aws.amazon.com/kms/latest/developerguide/kms-api-permissions-reference.html
// [ABAC for KMS]: https://docs.aws.amazon.com/kms/latest/developerguide/abac.html
// [KMS eventual consistency]: https://docs.aws.amazon.com/kms/latest/developerguide/programming-eventual-consistency.html
// [Controlling access to aliases]: https://docs.aws.amazon.com/kms/latest/developerguide/kms-alias.html#alias-access
func (c *Client) CreateAlias(ctx context.Context, params *CreateAliasInput, optFns ...func(*Options)) (*CreateAliasOutput, error) {
	if params == nil {
		params = &CreateAliasInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "CreateAlias", params, optFns, c.addOperationCreateAliasMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*CreateAliasOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type CreateAliasInput struct {

	// Specifies the alias name. This value must begin with alias/ followed by a name,
	// such as alias/ExampleAlias .
	//
	// Do not include confidential or sensitive information in this field. This field
	// may be displayed in plaintext in CloudTrail logs and other output.
	//
	// The AliasName value must be string of 1-256 characters. It can contain only
	// alphanumeric characters, forward slashes (/), underscores (_), and dashes (-).
	// The alias name cannot begin with alias/aws/ . The alias/aws/ prefix is reserved
	// for [Amazon Web Services managed keys].
	//
	// [Amazon Web Services managed keys]: https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#aws-managed-cmk
	//
	// This member is required.
	AliasName *string

	// Associates the alias with the specified [customer managed key]. The KMS key must be in the same
	// Amazon Web Services Region.
	//
	// A valid key ID is required. If you supply a null or empty string value, this
	// operation returns an error.
	//
	// For help finding the key ID and ARN, see [Finding the Key ID and ARN] in the Key Management Service
	// Developer Guide .
	//
	// Specify the key ID or key ARN of the KMS key.
	//
	// For example:
	//
	//   - Key ID: 1234abcd-12ab-34cd-56ef-1234567890ab
	//
	//   - Key ARN:
	//   arn:aws:kms:us-east-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab
	//
	// To get the key ID and key ARN for a KMS key, use ListKeys or DescribeKey.
	//
	// [customer managed key]: https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#customer-cmk
	// [Finding the Key ID and ARN]: https://docs.aws.amazon.com/kms/latest/developerguide/viewing-keys.html#find-cmk-id-arn
	//
	// This member is required.
	TargetKeyId *string

	noSmithyDocumentSerde
}

type CreateAliasOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationCreateAliasMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpCreateAlias{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpCreateAlias{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "CreateAlias"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addComputePayloadSHA256(stack); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addOpCreateAliasValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opCreateAlias(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opCreateAlias(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "CreateAlias",
	}
}
