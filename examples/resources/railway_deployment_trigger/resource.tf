resource "railway_deployment_trigger" "api" {
  project_id      = railway_project.example.id
  environment_id  = railway_project.example.default_environment.id
  service_id      = railway_service.example.id
  source_provider = "github"
  repository      = "your-org/your-repo"
  branch          = "main"
}
