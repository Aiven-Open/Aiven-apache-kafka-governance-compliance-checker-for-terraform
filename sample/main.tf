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


# Organization
resource "aiven_organization" "foo" {
  name = "My Organization"
}

# Users
resource "aiven_organization_user" "alice" {
  organization_id =  aiven_organization.foo.id
  user_email = "alice@aiven.fi"
}

resource "aiven_organization_user" "bob" {
  organization_id =  aiven_organization.foo.id
  user_email = "bob@aiven.fi"
}

resource "aiven_organization_user" "charlie" {
  organization_id =  aiven_organization.foo.id
  user_email = "charlie@aiven.fi"
}

resource "aiven_organization_user" "david" {
  organization_id =  aiven_organization.foo.id
  user_email = "david@aiven.fi"
}

resource "aiven_organization_user" "eve" {
  organization_id =  aiven_organization.foo.id
  user_email = "eve@aiven.fi"
}

# External identities
data "aiven_external_identity" "alice" {
  organization_id =  aiven_organization.foo.id
  internal_user_id =  aiven_organization_user.alice.user_id
  external_user_id = "alicia"
  external_service_name = "github"
}

data "aiven_external_identity" "bob" {
  organization_id =  aiven_organization.foo.id
  internal_user_id =  aiven_organization_user.bob.user_id
  external_user_id = "bobbie"
  external_service_name = "github"
}

data "aiven_external_identity" "charlie" {
  organization_id =  aiven_organization.foo.id
  internal_user_id =  aiven_organization_user.charlie.user_id
  external_user_id = "charlie-git"
  external_service_name = "github"
}

data "aiven_external_identity" "david" {
  organization_id =  aiven_organization.foo.id
  internal_user_id =  aiven_organization_user.david.user_id
  external_user_id = "david-git"
  external_service_name = "github"
}

data "aiven_external_identity" "eve" {
  organization_id =  aiven_organization.foo.id
  internal_user_id =  aiven_organization_user.eve.user_id
  external_user_id = "eve-github"
  external_service_name = "github"
}

# Groups
resource "aiven_organization_user_group" "foo" {
  description = "Example group of users."
  organization_id =  aiven_organization.foo.id
  name = "foo"
}

resource "aiven_organization_user_group" "bar" {
  description = "Example group of users."
  organization_id =  aiven_organization.foo.id
  name = "bar"
}

# Members
resource "aiven_organization_user_group_member" "alice-foo" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.foo.group_id
  user_id =  aiven_organization_user.alice.user_id
}

resource "aiven_organization_user_group_member" "bob-foo" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.foo.group_id
  user_id =  aiven_organization_user.bob.user_id
}

resource "aiven_organization_user_group_member" "charlie-foo" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.foo.group_id
  user_id =  aiven_organization_user.charlie.user_id
}

resource "aiven_organization_user_group_member" "david-foo" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.foo.group_id
  user_id =  aiven_organization_user.david.user_id
}

resource "aiven_organization_user_group_member" "eve-foo" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.foo.group_id
  user_id =  aiven_organization_user.eve.user_id
}

resource "aiven_organization_user_group_member" "alice-bar" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.bar.group_id
  user_id =  aiven_organization_user.alice.user_id
}

resource "aiven_organization_user_group_member" "bob-bar" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.bar.group_id
  user_id =  aiven_organization_user.bob.user_id
}

resource "aiven_organization_user_group_member" "charlie-bar" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.bar.group_id
  user_id =  aiven_organization_user.charlie.user_id
}

resource "aiven_organization_user_group_member" "david-bar" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.bar.group_id
  user_id =  aiven_organization_user.david.user_id
}

resource "aiven_organization_user_group_member" "eve-bar" {
  organization_id =  aiven_organization.foo.id
  group_id =  aiven_organization_user_group.bar.group_id
  user_id =  aiven_organization_user.eve.user_id
}

# Project
data "aiven_project" "foo" {
  project = "testproject-gl4h"
}

# Services
resource "aiven_kafka" "foo" {
  project                 = data.aiven_project.foo.project
  cloud_name              = "google-europe-west1"
  plan                    = "startup-2"
  service_name            = "kafka1"
  maintenance_window_dow  = "monday"
  maintenance_window_time = "10:00:00"
}

# Topics
resource "aiven_kafka_topic" "events" {
  project             = data.aiven_project.foo.project
  service_name        = aiven_kafka.foo.service_name
  topic_name          = "events"
  partitions          = 3
  replication         = 3
  owner_user_group_id = aiven_organization_user_group.bar.group_id
}

resource "aiven_kafka_topic" "void" {
  count = 3
  project             = data.aiven_project.foo.project
  service_name        = aiven_kafka.foo.service_name
  topic_name          = "void-${count.index}"
  partitions          = 3
  replication         = 2
  owner_user_group_id = aiven_organization_user_group.foo.group_id
}

resource "aiven_kafka_topic" "logs" {
  project             = data.aiven_project.foo.project
  service_name        = aiven_kafka.foo.service_name
  topic_name          = "logs"
  partitions          = 3
  replication         = 2
}

resource "aiven_kafka_topic" "logs2" {
  project             = data.aiven_project.foo.project
  service_name        = aiven_kafka.foo.service_name
  topic_name          = "logs2"
  partitions          = 3
  replication         = 2
  owner_user_group_id = aiven_organization_user_group.foo.group_id
}

resource "aiven_governance_subscription" "flip" {
  organization_id =  aiven_organization.foo.id
  subscription_name = "My Subscription"
  subscription_type = "KAFKA"
  owner_user_group_id = aiven_organization_user_group.foo.group_id
  subscription_data {
    project      = data.aiven_project.foo.project
    service_name = aiven_kafka.foo.service_name
	  username = "api1"
    acls {
      resource_name   = "foo"
      resource_type   = "Topic"
      operation       = "Write"
      permission_type = "ALLOW"
	    host            = "*"
    }
    acls {
      resource_name   = "bar"
      resource_type   = "Topic"
      operation       = "Read"
      permission_type = "ALLOW"
      host            = "*"
    }
  }
}