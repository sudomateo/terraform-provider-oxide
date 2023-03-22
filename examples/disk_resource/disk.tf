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

data "oxide_global_images" "image_example" {}

data "oxide_projects" "project_list" {}

resource "oxide_disk" "example" {
  project_id        = data.oxide_projects.project_list.projects.0.id
  description       = "a test disk"
  name              = "mydisk"
  size              = 1073741824
  disk_source       = { blank = 512 }
}

resource "oxide_disk" "example2" {
  project_id        = data.oxide_projects.project_list.projects.0.id
  description       = "a test disk"
  name              = "mydisk2"
  size              = 1073741824
  disk_source       = { global_image = data.oxide_global_images.image_example.global_images.0.id }
}