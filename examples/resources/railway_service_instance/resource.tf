resource "railway_service_instance" "api" {
  service_id     = railway_service.api.id
  environment_id = railway_project.example.default_environment.id

  source_repo    = "your-org/your-repo"
  root_directory = "apps/api"
  num_replicas   = 1
  region         = "us-east4-eqdc4a"
}
