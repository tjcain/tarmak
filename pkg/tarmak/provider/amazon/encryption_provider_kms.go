package amazon

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms"
)

func (a *Amazon) EncryptionProviderKMSName() string {
	return fmt.Sprintf("alias/tarmak/%s/encryption-provider", a.tarmak.Environment().Name())
}

func (a *Amazon) initEncrytionProviderKMS() error {
	svc, err := a.KMS()
	if err != nil {
		return err
	}

	k, err := svc.CreateKey(&kms.CreateKeyInput{
		Description: aws.String(fmt.Sprintf("KMS key for Tarmak provider's '%s' encryption provider", a.Name())),
		Tags: []*kms.Tag{
			{
				TagKey:   aws.String("provider"),
				TagValue: aws.String(a.Name()),
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = svc.CreateAlias(&kms.CreateAliasInput{
		TargetKeyId: aws.String(*k.KeyMetadata.KeyId),
		AliasName:   aws.String(a.EncryptionProviderKMSName()),
	})
	if err != nil {
		return err
	}

	a.encryptionProviderKMS = *k.KeyMetadata.Arn

	return nil
}

func (a *Amazon) ensureEncryptionProviderKMS() error {
	svc, err := a.KMS()
	if err != nil {
		return err
	}

	k, err := svc.DescribeKey(&kms.DescribeKeyInput{
		KeyId: aws.String(a.EncryptionProviderKMSName()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if strings.Contains(awsErr.Code(), "NotFound") {
				return a.initEncrytionProviderKMS()
			}
		}

		return fmt.Errorf("error looking for encryption provider kms alias: %s", err)
	}

	a.remoteStateKMS = *k.KeyMetadata.Arn

	return nil
}
