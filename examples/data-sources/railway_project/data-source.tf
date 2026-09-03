data "railway_project" "charming" {
  id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}

output "project_name" {
  value = data.railway_project.charming.name
}
