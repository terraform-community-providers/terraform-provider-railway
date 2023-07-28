resource "railway_custom_domain" "api" {
  repository     = "api.example.com"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}
