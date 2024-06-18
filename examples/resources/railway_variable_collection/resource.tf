resource "railway_variable_collection" "example" {
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id

  variables = {
    SENTRY_KEY    = "KEY"
    SENTRY_SECRET = "SECRET"
  }
}
