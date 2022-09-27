terraform {
  required_version = ">= 1.0"

  required_providers {
    oxide = {
      source  = "oxidecomputer/oxide"
      version = "0.1.0-dev"
    }
  }
}

provider "oxide" {}

resource "oxide_ip_pool" "example" {
  description = "a test IP pool"
  name        = "myippool"
  ranges {
    ip_version    = "ipv4"
    first_address = "172.20.15.227"
    last_address  = "172.20.15.239"
  }
}
