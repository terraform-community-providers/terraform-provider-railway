data "railway_project" "charming" {
  id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}

data "railway_environment" "production" {
  id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
}

data "railway_service" "server" {
  id = "39da7e07-fa3a-42fd-b695-d229319f2993"
}

data "railway_custom_domain" "api" {
  project_id     = data.railway_project.charming.id
  environment_id = data.railway_environment.production.id
  service_id     = data.railway_service.server.id
  domain         = "api.example.com"
}

output "custom_domain_dns" {
  value = {
    id                     = data.railway_custom_domain.api.id
    target_port            = data.railway_custom_domain.api.target_port
    cname_host             = data.railway_custom_domain.api.host_label
    zone                   = data.railway_custom_domain.api.zone
    cname_value            = data.railway_custom_domain.api.dns_record_value
    verification_txt_host  = data.railway_custom_domain.api.verification_host_label
    verification_txt_value = data.railway_custom_domain.api.verification_record_value
  }
}
