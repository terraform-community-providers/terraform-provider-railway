variable "railway_token" {
  type      = string
  sensitive = true
}

variable "railway_project_token" {
  type      = string
  sensitive = true
}

provider "railway" {
  token = var.railway_token
}

provider "railway" {
  alias         = "project"
  project_token = var.railway_project_token
}
