data "railway_project" "charming" {
  id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}

data "railway_environment" "production" {
  id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
}

output "environment_project" {
  value = {
    name       = data.railway_environment.production.name
    project_id = data.railway_project.charming.id
  }
}
