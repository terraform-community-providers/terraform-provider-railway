resource "railway_bucket" "uploads" {
  name           = "uploads"
  project_id     = railway_project.example.id
  environment_id = railway_environment.example.id
  region         = "ams"
}
