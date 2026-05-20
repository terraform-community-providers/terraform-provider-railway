resource "railway_service_domain" "api" {
  subdomain      = "example-api"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}

resource "railway_service_domain" "api_with_target_port" {
  subdomain      = "example-api-8000"
  target_port    = 8000
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}
