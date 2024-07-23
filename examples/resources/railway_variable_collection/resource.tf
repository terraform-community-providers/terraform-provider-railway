resource "railway_variable_collection" "example" {
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id

  variables = [
    {
      name  = "SENTRY_KEY"
      value = "KEY"
    },
    {
      name  = "SENTRY_SECRET"
      value = "SECRET"
    }
  ]
}
