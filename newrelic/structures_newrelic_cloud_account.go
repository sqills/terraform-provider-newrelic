package newrelic

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/newrelic/newrelic-client-go/pkg/cloud"
)

func serializeCloudLinkedAccountIds(ids []int) string {
	sort.Ints(ids)
	return serializeIDs(ids)
}

func resourceLinkedCloudAccountHash(v interface{}) int {
	var buf bytes.Buffer
	m, castOk := v.(map[string]interface{})
	if !castOk {
		return 0
	}
	if v, ok := m["linked_account_id"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["arn"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["aws_account_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["access_key_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["secret_access_key"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["application_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["client_secret"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["subscription_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["tenant_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["project_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return schema.HashString(buf.String())
}

func expandCloudAWSLinkAccountInputs(set *schema.Set) []cloud.CloudAwsLinkAccountInput {
	result := make([]cloud.CloudAwsLinkAccountInput, set.Len())
	for i, awsAccount := range set.List() {
		m := awsAccount.(map[string]interface{})
		result[i] = cloud.CloudAwsLinkAccountInput{
			Arn:  m["arn"].(string),
			Name: m["name"].(string),
		}
	}

	return result
}

func expandCloudUnlinkAccountInputs(set *schema.Set) []cloud.CloudUnlinkAccountsInput {
	result := make([]cloud.CloudUnlinkAccountsInput, set.Len())
	for i, account := range set.List() {
		m := account.(map[string]interface{})
		result[i] = cloud.CloudUnlinkAccountsInput{
			LinkedAccountId: m["linked_account_id"].(int),
		}
	}

	return result
}

func flattenLinkedCloudAccounts(accountID int, linkedAccounts []cloud.CloudLinkedAccount, d *schema.ResourceData) error {
	var linkedAccountIds []int
	var aws []interface{}

	for _, linkedAccount := range linkedAccounts {
		linkedAccountIds = append(linkedAccountIds, linkedAccount.ID)

		switch linkedAccount.Provider.(type) {
		case *cloud.CloudAwsProvider:
			aws = append(aws, map[string]interface{}{
				"linked_account_id": linkedAccount.ID,
				"name":              linkedAccount.Name,
				"arn":               linkedAccount.AuthLabel,
			})
		default:
			return fmt.Errorf("got a linked account for an unknown provider")
		}
	}

	d.Set("account_id", accountID)
	d.Set("aws", aws)
	d.SetId(serializeCloudLinkedAccountIds(linkedAccountIds))

	return nil
}
