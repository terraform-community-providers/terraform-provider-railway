resource "railway_service" "example" {
  name       = "api"
  project_id = railway_project.example.id
}

resource "railway_service" "with_environment" {
  name           = "api"
  project_id     = railway_project.example.id
  environment_id = railway_environment.staging.id
}
