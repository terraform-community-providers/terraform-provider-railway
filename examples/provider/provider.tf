provider "railway" {
  # Authentication (can also use RAILWAY_TOKEN environment variable)
  # token = var.railway_token

  # Retry configuration (optional - defaults shown)
  # max_retries        = 5
  # initial_backoff_ms = 1000
  # max_backoff_ms     = 30000

  # Proactive rate limiting (optional - disabled by default)
  # rate_limit_rps = 2.0
}
