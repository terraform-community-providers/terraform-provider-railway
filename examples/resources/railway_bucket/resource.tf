resource "railway_bucket" "uploads" {
  name           = "uploads"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
  region         = "sjc"
}
