resource "railway_environment" "example" {
  name       = "staging"
  project_id = railway_project.example.id
}

resource "railway_environment" "clone" {
  name                  = "review"
  project_id            = railway_project.example.id
  source_environment_id = railway_environment.example.id
}
