# Read variables from a Postgres service to use in app configuration
data "railway_service_variables" "postgres" {
  service_id     = railway_service.postgres.id
  environment_id = railway_project.example.default_environment.id
}

# Use the DATABASE_URL from Postgres in the app service
resource "railway_variable" "app_database_url" {
  name           = "DATABASE_URL"
  value          = data.railway_service_variables.postgres.variables["DATABASE_URL"]
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.app.id
}
