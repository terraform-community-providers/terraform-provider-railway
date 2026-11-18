resource "railway_access_group" "example" {
  workspace_id = railway_project.example.workspace_id
  name         = "platform"
  role         = "MEMBER"
  member_ids   = ["89fa0236-2b1b-4a8c-b12d-ae3634b30d97"]
  project_ids  = [railway_project.example.id]
}
