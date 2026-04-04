resource "railway_service" "example" {
  name       = "api"
  project_id = railway_project.example.id

  redeploy_environment_ids = toset([
    railway_project.example.default_environment.id,
  ])
}
