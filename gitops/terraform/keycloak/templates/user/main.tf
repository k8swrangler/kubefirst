terraform {
  required_providers {
    keycloak = {
      source  = "mrparkers/keycloak"
      version = ">= 2.0.0"
    }
  }
}

provider "keycloak" {
    client_id     = "admin-cli"
    username      = "gitlab-bot"
    password      = "ATU6VaGr6A"
    url           = "https://keycloak.<AWS_HOSTED_ZONE_NAME>"
}


variable "first_name" {
  type = string
}

variable "last_name" {
  type = string
}

variable "username" {
  type = string
}

variable "realm_id" {
  type    = string
  default = "kubefirst"
}

variable "store_password" {
  type = bool
}

output "username" {
  value = keycloak_user.user.username
}
