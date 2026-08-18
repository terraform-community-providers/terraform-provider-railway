resource "railway_bucket" "uploads" {
  name       = "uploads"
  project_id = railway_project.example.id
}
