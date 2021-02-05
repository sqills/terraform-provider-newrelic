package newrelic

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/cloud"
	nrerrors "github.com/newrelic/newrelic-client-go/pkg/errors"
)

func resourceNewRelicCloudAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceNewRelicCloudAccountCreate,
		Read:   resourceNewRelicCloudAccountRead,
		Update: resourceNewRelicCloudAccountUpdate,
		Delete: resourceNewRelicCloudAccountDelete,
		Importer: &schema.ResourceImporter{
			State: resourceImportStateWithMetadata(2, "type"),
		},
		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The New Relic account ID where the Cloud accounts will be linked to",
			},
			"aws": {
				Type:        schema.TypeSet,
				Set:         resourceLinkedCloudAccountHash,
				ConfigMode:  schema.SchemaConfigModeAttr,
				Description: "Link a New Relic account to one or more AWS cloud accounts",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"linked_account_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the linked cloud account in New Relic",
						},
						"arn": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The AWS role ARN (used to fetch data)",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The linked account name",
						},
					},
				},
			},
			//"aws_govcloud": {
			//	Type:        schema.TypeSet,
			//	Set:         resourceLinkedCloudAccountHash,
			//	Description: "Link a New Relic account to one or more AWS Govcloud accounts",
			//	Elem: &schema.Resource{
			//		Schema: map[string]*schema.Schema{
			//			"access_key_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The key used to make requests to AWS service APIs",
			//			},
			//			"aws_account_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The AWS account id",
			//			},
			//			"name": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The linked account name",
			//			},
			//			"secret_access_key": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The secret key used to make requests to AWS service APIs",
			//			},
			//		},
			//	},
			//},
			//"azure": {
			//	Type:        schema.TypeSet,
			//	Set:         resourceLinkedCloudAccountHash,
			//	Description: "Link a New Relic account to one or more Azure cloud accounts",
			//	Elem: &schema.Resource{
			//		Schema: map[string]*schema.Schema{
			//			"application_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The Azure account application identifier (used to fetch data)",
			//			},
			//			"client_secret": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The Azure account application secret key",
			//			},
			//			"name": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The linked account name",
			//			},
			//			"subscription_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The Azure account subscription identifier",
			//			},
			//			"tenant_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The Azure account tenant identifier",
			//			},
			//		},
			//	},
			//},
			//"gcp": {
			//	Type:        schema.TypeSet,
			//	Set:         resourceLinkedCloudAccountHash,
			//	Description: "Link a New Relic account to one or more GCP cloud accounts",
			//	Elem: &schema.Resource{
			//		Schema: map[string]*schema.Schema{
			//			"name": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The linked account name",
			//			},
			//			"project_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The GCP project identifier",
			//			},
			//			"tenant_id": {
			//				Type:        schema.TypeString,
			//				Required:    true,
			//				Description: "The Azure account tenant identifier",
			//			},
			//		},
			//	},
			//},
		},
	}
}

func resourceNewRelicCloudAccountCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	accountID := selectAccountID(providerConfig, d)

	aws := d.Get("aws").(*schema.Set)
	linkedAccountIds, err := cloudLinkAccount(client, accountID, cloud.CloudLinkCloudAccountsInput{
		Aws: expandCloudAWSLinkAccountInputs(aws),
	})
	if err != nil {
		return err
	}
	d.SetId(serializeCloudLinkedAccountIds(linkedAccountIds))

	return resourceNewRelicCloudAccountRead(d, meta)
}

func resourceNewRelicCloudAccountRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	accountID := selectAccountID(providerConfig, d)

	log.Printf("[INFO] Reading New Relic Linked Cloud Accounts %s", d.Id())

	linkedAccountIds, err := parseHashedIDs(d.Id())
	if err != nil {
		return err
	}
	linkedAccountIdSet := make(map[int]bool, len(linkedAccountIds))
	for _, id := range linkedAccountIds {
		linkedAccountIdSet[id] = true
	}

	var linkedAccounts []cloud.CloudLinkedAccount
	for _, provider := range []string{"aws"} {
		linkedAccountsForProvider, err := client.Cloud.GetLinkedAccounts(provider)
		if err != nil {
			if _, ok := err.(*nrerrors.NotFound); ok {
				continue
			}
			return err
		}

		for _, linkedAccount := range *linkedAccountsForProvider {
			if linkedAccountIdSet[linkedAccount.ID] {
				linkedAccounts = append(linkedAccounts, linkedAccount)
			}
		}
	}

	return flattenLinkedCloudAccounts(accountID, linkedAccounts, d)
}

func resourceNewRelicCloudAccountUpdate(d *schema.ResourceData, meta interface{}) error {
	linkedAccountIds, err := parseHashedIDs(d.Id())
	if err != nil {
		return err
	}

	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	accountID := selectAccountID(providerConfig, d)

	var unlinkInputs []cloud.CloudUnlinkAccountsInput
	var linkInput cloud.CloudLinkCloudAccountsInput

	if d.HasChange("aws") {
		o, n := d.GetChange("aws")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		if unlink := os.Difference(ns); unlink.Len() > 0 {
			unlinkInputs = append(unlinkInputs, expandCloudUnlinkAccountInputs(unlink)...)
		}
		if link := ns.Difference(os); link.Len() > 0 {
			linkInput.Aws = expandCloudAWSLinkAccountInputs(link)
		}
	}

	if len(unlinkInputs) > 0 {
		unlinkedAccountIds, err := cloudUnlinkAccount(client, accountID, unlinkInputs)
		if err != nil {
			return err
		}
		linkedAccountIds = removeIds(linkedAccountIds, unlinkedAccountIds)
	}

	if len(linkInput.Aws) > 0 || len(linkInput.AwsGovcloud) > 0 || len(linkInput.Azure) > 0 || len(linkInput.Gcp) > 0 {
		addedLinkedAccountIds, err := cloudLinkAccount(client, accountID, linkInput)
		if err != nil {
			return err
		}
		linkedAccountIds = append(linkedAccountIds, addedLinkedAccountIds...)
	}

	d.SetId(serializeCloudLinkedAccountIds(linkedAccountIds))

	return resourceNewRelicCloudAccountRead(d, meta)
}

func resourceNewRelicCloudAccountDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	accountID := selectAccountID(providerConfig, d)

	linkedAccountIds, err := parseHashedIDs(d.Id())
	if err != nil {
		return err
	}

	inputs := make([]cloud.CloudUnlinkAccountsInput, len(linkedAccountIds))
	for i, linkedAccountId := range linkedAccountIds {
		inputs[i] = cloud.CloudUnlinkAccountsInput{LinkedAccountId: linkedAccountId}
	}

	if _, err := cloudUnlinkAccount(client, accountID, inputs); err != nil {
		return err
	}

	return nil
}

func cloudLinkAccount(client *newrelic.NewRelic, accountID int, input cloud.CloudLinkCloudAccountsInput) ([]int, error) {
	payload, err := client.Cloud.CloudLinkAccount(accountID, input)
	if err != nil {
		return nil, err
	}
	if len(payload.Errors) > 0 {
		return nil, fmt.Errorf(payload.Errors[0].Message)
	}

	linkedAccountIds := make([]int, len(payload.LinkedAccounts))
	for i, linkedAccount := range payload.LinkedAccounts {
		linkedAccountIds[i] = linkedAccount.ID
	}

	return linkedAccountIds, nil
}

func cloudUnlinkAccount(client *newrelic.NewRelic, accountID int, input []cloud.CloudUnlinkAccountsInput) ([]int, error) {
	payload, err := client.Cloud.CloudUnlinkAccount(accountID, input)
	if err != nil {
		return nil, err
	}
	if len(payload.Errors) > 0 {
		return nil, fmt.Errorf(payload.Errors[0].Message)
	}
	unlinkedAccountIds := make([]int, len(payload.UnlinkedAccounts))
	for i, linkedAccount := range payload.UnlinkedAccounts {
		unlinkedAccountIds[i] = linkedAccount.ID
	}

	return unlinkedAccountIds, nil
}
