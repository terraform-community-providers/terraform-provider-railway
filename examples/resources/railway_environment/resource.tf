resource "railway_environment" "example" {
  name       = "staging"
  project_id = railway_project.example.id
}
