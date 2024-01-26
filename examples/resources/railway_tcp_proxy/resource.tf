resource "railway_tcp_proxy" "redis" {
  application_port = 6379
  environment_id   = railway_project.example.default_environment.id
  service_id       = railway_service.example.id
}
