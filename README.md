![test](https://github.com/aiven/aiven-terraform-governance-compliance-checker/actions/workflows/test.yml/badge.svg)
![lint](https://github.com/aiven/aiven-terraform-governance-compliance-checker/actions/workflows/lint.yml/badge.svg)
![codeql](https://github.com/aiven/aiven-terraform-governance-compliance-checker/actions/workflows/codeql.yml/badge.svg)

## Overview
This GitHub Action can be used to perform governing checks on terraform aiven provider resources for the terraform generated plan. 
It outputs a compliance report in JSON format with any errors it finds.

Example report:
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


## Example
This workflow gets the requester and approvers from the current pull request and uses the action to check the plan compliance during pull request reviews:
```yaml
name: 'Check plan'

on:
  pull_request:
    types:
      - opened
      - synchronize
      - reopened
      - labeled
      - unlabeled

  pull_request_review:
    types: [submitted, dismissed, edited]
    
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - name: "Checkout branch"
        uses: actions/checkout@v4

      - name: "Pull request reviewers"
        id: pull_request_reviewers
        uses: octokit/request-action@v2.x
        with:
          route: GET /repos/${{ github.repository }}/pulls/${{ github.event.pull_request.number }}/reviews
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: "Pull request approvers"
        id: "pull_request_approvers"
        run: |
          APPROVERS=$(
            echo '${{ steps.pull_request_reviewers.outputs.data }}' | jq '[.[] | select(.state == "APPROVED") | .user.login] | unique | @csv' | tr -d \"
          )
          echo "approvers=$APPROVERS" >> "$GITHUB_OUTPUT"
        shell: bash

      - name: "Setup terraform"
        uses: hashicorp/setup-terraform@v3

      - name: "Terraform plan"
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_KEY }}
          PROVIDER_AIVEN_ENABLE_BETA: 1
        run: |
          terraform init
          terraform plan -out=./plan -var="aiven_api_token=${{ secrets.AIVEN_API_TOKEN }}"
          terraform show -json ./plan > ./plan.json
        shell: bash

      - name: "Run compliance check"
        id: "governance"
        uses: aiven/aiven-terraform-governance-compliance-checker@42d0bff4571d8ff79cc8bbcece855659f50b00c8
        with:
          requester: ${{ github.event.pull_request.user.login }}
          approvers: ${{ steps.pull_request_approvers.outputs.approvers }}
          plan: "./plan.json"

      - name: Comment OK Report on PR
        id: comment-ok
        if: ${{ fromJson(steps.governance.outputs.result).ok == true }}
        uses: thollander/actions-comment-pull-request@v2
        with:
          message: |
            ### Compliance report: ✅
          pr_number: ${{ github.event.pull_request.number }}
          comment_tag: compliance

      - name: Comment NOK Report on PR
        id: comment-nok
        if: ${{ fromJson(steps.governance.outputs.result).ok == false }}
        uses: thollander/actions-comment-pull-request@v2
        with:
          message: |
            ### Compliance report:
            ```json
            ${{ toJson(fromJson(steps.governance.outputs.result)) }}
            ```
          pr_number: ${{ github.event.pull_request.number }}
          comment_tag: compliance
```


