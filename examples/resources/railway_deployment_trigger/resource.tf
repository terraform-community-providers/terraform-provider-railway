resource "railway_deployment_trigger" "example" {
  repository     = "railwayapp/railway-example-nodejs"
  branch         = "main"
  check_suites   = true
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}
