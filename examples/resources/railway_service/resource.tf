resource "railway_service" "example" {
  name       = "api"
  project_id = railway_project.example.id
}
