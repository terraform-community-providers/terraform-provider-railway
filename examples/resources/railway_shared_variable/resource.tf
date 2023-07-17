resource "railway_shared_variable" "example" {
  name           = "SENTRY_KEY"
  value          = "1234567890"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
}
