data "railway_project" "charming" {
  id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}

data "railway_environment" "production" {
  id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
}

data "railway_service" "server" {
  id = "39da7e07-fa3a-42fd-b695-d229319f2993"
}

data "railway_service_domain" "server" {
  project_id     = data.railway_project.charming.id
  environment_id = data.railway_environment.production.id
  service_id     = data.railway_service.server.id
}

output "generated_domain" {
  value = data.railway_service_domain.server.domain
}
