package newrelic

import (
	"testing"

	"github.com/newrelic/newrelic-client-go/pkg/cloud"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"github.com/stretchr/testify/require"
)

func TestSerializeCloudLinkedAccountIds(t *testing.T) {
	require.Equal(t, serializeCloudLinkedAccountIds([]int{1, 4, 6}), serializeCloudLinkedAccountIds([]int{1, 4, 6}))
	require.Equal(t, serializeCloudLinkedAccountIds([]int{1, 4, 6}), serializeCloudLinkedAccountIds([]int{4, 1, 6}))
	require.NotEqual(t, serializeCloudLinkedAccountIds([]int{1, 6}), serializeCloudLinkedAccountIds([]int{4, 1, 6}))
	require.NotEqual(t, serializeCloudLinkedAccountIds([]int{1}), serializeCloudLinkedAccountIds([]int{4}))
}

func TestExpandCloudAWSLinkAccountInputs(t *testing.T) {
	s := schema.NewSet(resourceLinkedCloudAccountHash, nil)
	inputs := expandCloudAWSLinkAccountInputs(s)
	require.Empty(t, inputs)

	s.Add(map[string]interface{}{
		"name": "Foo",
		"arn":  "foo",
	})
	s.Add(map[string]interface{}{
		"name": "Bar",
		"arn":  "bar",
	})
	inputs = expandCloudAWSLinkAccountInputs(s)
	require.Len(t, inputs, 2)
	require.Contains(t, inputs, cloud.CloudAwsLinkAccountInput{Name: "Foo", Arn: "foo"})
	require.Contains(t, inputs, cloud.CloudAwsLinkAccountInput{Name: "Bar", Arn: "bar"})
}

func TestExpandCloudUnlinkAccountInputs(t *testing.T) {
	s := schema.NewSet(resourceLinkedCloudAccountHash, nil)
	inputs := expandCloudUnlinkAccountInputs(s)
	require.Empty(t, inputs)

	s.Add(map[string]interface{}{
		"linked_account_id": 5,
	})
	s.Add(map[string]interface{}{
		"linked_account_id": 2,
	})
	inputs = expandCloudUnlinkAccountInputs(s)
	require.Len(t, inputs, 2)
	require.Contains(t, inputs, cloud.CloudUnlinkAccountsInput{LinkedAccountId: 5})
	require.Contains(t, inputs, cloud.CloudUnlinkAccountsInput{LinkedAccountId: 2})
}

func TestFlattenLinkedCloudAccounts(t *testing.T) {
	d := resourceNewRelicCloudAccount().TestResourceData()
	require.NoError(t, flattenLinkedCloudAccounts(123, []cloud.CloudLinkedAccount{
		{
			ID:        6,
			Name:      "X",
			AuthLabel: "iam_role_arn_for_x",
			Provider:  &cloud.CloudAwsProvider{},
		},
		{
			ID:        8,
			Name:      "Y",
			AuthLabel: "iam_role_arn_for_y",
			Provider:  &cloud.CloudAwsProvider{},
		},
	}, d))

	expectedAWS := []interface{}{
		map[string]interface{}{
			"linked_account_id": 6,
			"name":              "X",
			"arn":               "iam_role_arn_for_x",
		},
		map[string]interface{}{
			"linked_account_id": 8,
			"name":              "Y",
			"arn":               "iam_role_arn_for_y",
		},
	}

	require.Equal(t, "6:8", d.Id())
	require.Equal(t, 123, d.Get("account_id"))
	require.Equal(t, expectedAWS, d.Get("aws").(*schema.Set).List())
}
