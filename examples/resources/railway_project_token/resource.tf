resource "railway_project_token" "example" {
  name           = "example"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
}
