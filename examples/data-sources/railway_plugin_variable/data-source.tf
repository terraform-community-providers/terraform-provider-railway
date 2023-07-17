data "railway_plugin_variable" "example" {
  name           = "REDIS_URL"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
  plugin_id      = railway_plugin.example.id
}
