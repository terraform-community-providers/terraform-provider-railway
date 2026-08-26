ephemeral "railway_bucket_credentials" "uploads" {
  project_id     = railway_bucket.uploads.project_id
  environment_id = railway_bucket.uploads.environment_id
  bucket_id      = railway_bucket.uploads.id
}
