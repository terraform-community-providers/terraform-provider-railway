resource "railway_service_instance" "api" {
  environment_id   = railway_environment.preview.id
  service_id       = railway_service.api.id
  source_image     = "traefik/whoami:v1.10"
  healthcheck_path = "/"
  vcpus            = 0.5
  memory_gb        = 1
}
