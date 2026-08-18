resource "railway_bucket" "uploads" {
  name       = "uploads"
  project_id = railway_project.example.id
}

resource "railway_bucket_instance" "uploads" {
  bucket_id      = railway_bucket.uploads.id
  environment_id = railway_environment.example.id
  region         = "ams"
}

resource "railway_bucket_cors_configuration" "uploads" {
  project_id     = railway_project.example.id
  environment_id = railway_bucket_instance.uploads.environment_id
  bucket_id      = railway_bucket_instance.uploads.bucket_id

  cors_rules = [
    {
      id              = "browser-uploads"
      allowed_headers = ["*"]
      allowed_methods = ["GET", "HEAD", "PUT"]
      allowed_origins = ["https://example.com"]
      expose_headers  = ["ETag"]
      max_age_seconds = 3600
    },
  ]
}
