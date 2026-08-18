resource "railway_bucket_instance" "uploads" {
  bucket_id      = railway_bucket.uploads.id
  environment_id = railway_environment.example.id
  region         = "ams"
}
