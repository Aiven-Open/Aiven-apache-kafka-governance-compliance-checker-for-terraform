## Build the plugin
Build the plugin from the root folder, so we can use it for testing.

### Prerequisetes
* Go version 1.23 or higher
* Terraform
* Make
* [Aiven API Token](https://aiven.io/docs/platform/howto/create_authentication_token)
### Build
make build

### Tests
make test

## Test the plugin locally
Using the sample folder

### Add a project in your Aiven Console project
main.tf has a project called "testproject-gl4h" either create an empty project by the same name or substitute in an existing project you have in Aiven Console.

### Enable aiven_external_resource
the Aiven external resource links your github account to the aiven user accounts.
export PROVIDER_AIVEN_ENABLE_BETA=1

### Create the Terraform plan
You will need to set your Aiven API token you created earlier in the Prerequisetes steps
#### Setup your API Token
export API_TOKEN="Your token"
export AIVEN_WEB_TOKEN="Your token"
#### Setup your target deployment
export AIVEN_WEB_URL=https://aiven.io

execute the following Terraform commands from inside the samples folder
From the 'samples' folder
Initialise Terraform

``terraform init``

Create a Terraform Plan

``terraform plan -var="aiven_api_token=${API_TOKEN}" -out=plan``

Mutate terraform plan to Json

``terraform show -json ./plan > ./plan.json``

Test Plan with the checker (no approvers)

``../build/checker -plan=plan.json -requester=charlie -approvers=``

Test Plan with the checker with approvers

``../build/checker -plan=plan.json -requester=charlie-git -approvers=david,bobbie``





## Using a custom Terraform build

Create the folder ``$HOME/.terraformrc``

Now we must add the custom provider override in this folder edit the file and add

```
provider_installation {
  dev_overrides {
     "registry.terraform.io/aiven/aiven" = "/Users/<username>/.terraform.d/plugins/registry.terraform.io/aiven-dev/aiven/0.0.0+dev/linux_arm64/"
  }

  direct {}
}
```

Note:  Adjust the path accoring to your operating system and architecture