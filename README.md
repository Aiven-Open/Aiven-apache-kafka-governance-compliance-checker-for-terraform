# aiven-terraform-governance-compliance-checker
GitHub Action Workflow to check whether the plan generated by terraform plan complies with governance rules on aiven terraform provider resources.

# How it works
When a pull request is opened, the action will generate the plan using `terraform plan` and analyze the output to see
whether the user requesting the change is allowed to make the change (is a member of the owner group) and whether the pull request has been approved by a member of the owner group.

If the plan complies with governance rules, the workflow is considered succesful and no report is produced.

If the plan does **NOT** comply with governance rules, the action will produce a report and the workflow will be considered as failed. For example:
```json
{
  "ok": false,
  "errors": [
    {
      "error": "requesting user is not a member of the owner group",
      "address": "aiven_kafka_topic.foo"
    },
    {
      "error": "approval required from a member of the owner group",
      "address": "aiven_kafka_topic.foo"
    }
  ] 
}
```


# Setup
Here is how to set it up for your repository:
1. For the action to understand who is who, you'll need to map the github login username to aiven internal user using the `aiven_external_identity` data resource. For example:
```
data "aiven_external_identity" "foo" {
  organization_id = data.aiven_organization_user.organization_id,
  internal_user_id = data.aiven_organization_user.user_id,
  external_user_id = "github-username",
  external_service_name = "github"
}
```

2. The action only considers topics with owner, which you can define using `aiven_kafka_topic.owner_user_group_id`. For example:
```
resource "aiven_kafka_topic" "foo" {
  topic_name = "foo"
  owner_user_group_id = aiven_organization_user_group.foo.group_id
  ...
}
```

3. Finally, add the github action workflow into your `.github/workflows`


