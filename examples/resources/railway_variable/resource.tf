resource "railway_variable" "example" {
  name           = "SENTRY_KEY"
  value          = "1234567890"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}
