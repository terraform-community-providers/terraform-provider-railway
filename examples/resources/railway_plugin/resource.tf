resource "railway_plugin" "example" {
  name       = "notifications-db"
  type       = "redis"
  project_id = railway_project.example.id
}
