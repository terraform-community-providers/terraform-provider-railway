resource "railway_environment" "example" {
  name       = "staging"
  project_id = railway_project.example.id
}

resource "railway_environment" "cloned_preview" {
  name                  = "preview"
  project_id            = railway_project.example.id
  source_environment_id = railway_project.example.default_environment.id
  skip_initial_deploys  = true
}
