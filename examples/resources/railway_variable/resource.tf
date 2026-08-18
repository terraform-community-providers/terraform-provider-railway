resource "railway_variable" "example" {
  name           = "SENTRY_KEY"
  value          = "1234567890"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}

resource "railway_variable" "sealed" {
  name             = "API_KEY"
  value_wo         = var.api_key
  value_wo_version = var.api_key_version
  environment_id   = railway_project.example.default_environment.id
  service_id       = railway_service.example.id
}
