resource "railway_service_edge_rules" "example" {
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id

  rules = jsonencode({
    version = 1
    rules = [
      {
        description = "Block known bad IP ranges"
        priority    = 10
        enabled     = true
        if          = { attr = "ipv4.src", op = "in", value = ["192.0.2.0/24"] }
        then        = { action = "block", params = { status = 403 } }
      }
    ]
  })
}
