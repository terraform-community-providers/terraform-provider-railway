data "railway_workspace_members" "example" {
  workspace_id = railway_project.example.workspace_id
}

# Look up users by email, e.g. in a module that takes a list of emails.
resource "railway_access_group" "from_emails" {
  workspace_id = data.railway_workspace_members.example.workspace_id
  name         = "platform"
  role         = "MEMBER"
  member_ids   = [for email in var.member_emails : data.railway_workspace_members.example.ids_by_email[email]]
}
