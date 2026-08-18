resource "railway_volume_instance" "data" {
  volume_id      = railway_volume.data.id
  environment_id = railway_environment.production.id
  size_mb        = 5000
  service_id     = railway_service.api.id
  mount_path     = "/app/data"
}
