locals {
  github_repository_id = "828740035"
  github_owner_id      = "50233866"
}

resource "google_project_service" "wif" {
  for_each = toset([
    "iam.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
  ])

  service            = each.value
  disable_on_destroy = false
}

resource "google_iam_workload_identity_pool" "github" {
  depends_on = [google_project_service.wif]

  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id = google_iam_workload_identity_pool.github.workload_identity_pool_id

  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  attribute_mapping = {
    "google.subject"                = "assertion.sub"
    "attribute.actor"               = "assertion.actor"
    "attribute.repository_id"       = "assertion.repository_id"
    "attribute.repository_owner_id" = "assertion.repository_owner_id"
    "attribute.ref"                 = "assertion.ref"
    "attribute.event_name"          = "assertion.event_name"
  }

  attribute_condition = join(" && ", [
    "assertion.repository_id == '${local.github_repository_id}'",
    "assertion.repository_owner_id == '${local.github_owner_id}'",
    "assertion.ref == 'refs/heads/main'",
    "assertion.event_name == 'push'",
  ])
}

resource "google_service_account" "image_publisher" {
  depends_on = [google_project_service.wif]

  account_id   = "tasks-image-publisher"
  display_name = "Tasks image publisher"
}

resource "google_service_account_iam_member" "github_image_publisher" {
  service_account_id = google_service_account.image_publisher.name
  role               = "roles/iam.workloadIdentityUser"

  member = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository_id/${local.github_repository_id}"
}

resource "google_artifact_registry_repository_iam_member" "image_publisher" {
  project    = google_artifact_registry_repository.main.project
  location   = google_artifact_registry_repository.main.location
  repository = google_artifact_registry_repository.main.repository_id

  role   = "roles/artifactregistry.writer"
  member = "serviceAccount:${google_service_account.image_publisher.email}"
}

output "wif_provider" {
  value = google_iam_workload_identity_pool_provider.github.name
}

output "image_publisher_service_account" {
  value = google_service_account.image_publisher.email
}
