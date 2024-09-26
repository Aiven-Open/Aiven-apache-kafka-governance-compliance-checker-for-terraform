# aiven-terraform-governance-compliance-checker
Check whether terraform plan complies with aiven terraform provider resources.
https://developer.hashicorp.com/terraform/internals/json-format

# Usage
This program can be used for example in a GitHub Action to indicate if the 
plan generated by terraform complies with aiven terraform provider resources.

The plan is to have the GitHub action that wraps this code shared privately with customer repository,
which they can then choose to use as a validation step in their pipeline.
https://docs.github.com/en/actions/sharing-automations/sharing-actions-and-workflows-from-your-private-repository#about-github-actions-access-to-private-repositories

If we ever decide to make this repository public it could be published in
the github marketplace for wider audience.

# TODO
- test e2e with the external_identity resource
- the github action code

# How to run
```bash
terraform plan -out="./plan"
terraform show -json ./plan > ./plan.json

go run . -plan="./data/plan.json" -requester="alice" -approvers="bob,charlie

OR build first
go build
./checker -plan="./data/plan.json" -requester="alice" -approvers="bob,charlie
```
[comment]: <> (go run main.go -plan="./plan.json" -requester="alice" -approvers="bob,charlie")
[comment]: <> (doesn't work for multi-file go projects?)

# Example output
```json
{
  "ok": false,
  "messages": [
    {
      "title": "MembershipRequired",
      "description": "requester is not a member of the owner user group",
      "resource_type": "aiven_kafka_topic",
      "resource_name": "topic-1"
    },
    {
      "title": "ApprovalRequired",
      "description": "approval is required from the owner user group",
      "resource_type": "aiven_kafka_topic",
      "resource_name": "topic-1"
    }
  ] 
}
```

# How to run as a microservice
```bash
go run . -micro

OR build first 
go build
./checker -micro

3. Create a HTTP POST request to /check with curl or Postman etc.
```