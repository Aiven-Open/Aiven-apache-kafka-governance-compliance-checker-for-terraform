terraform {
  required_providers {
    aiven = {
      source = "aiven/aiven"
      version = ">= 4.0.0, < 5.0.0"
    }
  }
}

variable "aiven_api_token" {}

provider "aiven" {
  api_token = var.aiven_api_token
}

data "aiven_organization" "foo" {
  name = "My Organization"
}

data "aiven_organization_user" "foo" {
  organization_id = data.aiven_organization.foo.id
  user_email = "roope-kar@aiven.fi"
}

resource "aiven_organization_user_group" "foo" {
  description = "Example group of users."
  organization_id = data.aiven_organization.foo.id
  name = "foo"
}

resource "aiven_organization_user_group_member" "foo" {
  organization_id = data.aiven_organization.foo.id
  group_id = aiven_organization_user_group.foo.group_id
  user_id = data.aiven_organization_user.foo.user_id
}

data "aiven_project" "foo" {
  project = "testproject-r63c"
}

resource "aiven_kafka" "foo" {
  project                 = data.aiven_project.foo.project
  cloud_name              = "google-europe-west1"
  plan                    = "startup-2"
  service_name            = "kafka1"
  maintenance_window_dow  = "monday"
  maintenance_window_time = "10:00:00"
}

resource "aiven_kafka_topic" "foo" {
  project             = data.aiven_project.foo.project
  service_name        = aiven_kafka.foo.service_name
  topic_name          = "topic"
  partitions          = 3
  replication         = 2
  owner_user_group_id = aiven_organization_user_group.foo.group_id
}

data "aiven_external_identity" "alice" {
  organization_id = data.aiven_organization.foo.id
  internal_user_id = data.aiven_organization_user.foo.user_id
  external_user_id = "alice"
  external_service_name = "github"
}