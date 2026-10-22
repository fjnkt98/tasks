# Main
terraform {
  required_version = "~>1.17.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "8.5.0"
    }
  }

  backend "gcs" {
    bucket = "tasks-terraform-state"
    prefix = "tasks/production"
  }
}

provider "google" {
  project = "tasks-510111"
  region  = "asia-northeast1"
  zone    = "asia-northeast1-a"
}

locals {
  image_tag = "00de5beb2d8270617d579c3931f15076f25d6916"
}

# State
resource "google_storage_bucket" "terraform-state" {
  name     = "tasks-terraform-state"
  location = "asia-northeast1"

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Storage
resource "google_storage_bucket" "db" {
  name          = "tasks-db"
  location      = "asia-northeast1"
  force_destroy = false

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  lifecycle {
    prevent_destroy = true
  }
}

# Artifact Registry
resource "google_project_service" "artifact_registry" {
  service = "artifactregistry.googleapis.com"

  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "main" {
  depends_on = [google_project_service.artifact_registry]

  location      = "asia-northeast1"
  repository_id = "tasks"
  format        = "DOCKER"

  docker_config {
    immutable_tags = true
  }
}

# Secret Manager
resource "google_project_service" "secretmanager_api" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_secret_manager_secret" "admin_username" {
  depends_on = [google_project_service.secretmanager_api]

  secret_id = "tasks-admin-username"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "secretaccess_admin_username" {
  secret_id = google_secret_manager_secret.admin_username.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.app.email}"
}

resource "google_secret_manager_secret" "admin_password" {
  depends_on = [google_project_service.secretmanager_api]

  secret_id = "tasks-admin-password"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "secretaccess_admin_password" {
  secret_id = google_secret_manager_secret.admin_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.app.email}"
}

# Service Account
resource "google_service_account" "app" {
  account_id = "tasks-app"
}

resource "google_storage_bucket_iam_member" "app_storage" {
  bucket = google_storage_bucket.db.name
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.app.email}"
}

resource "google_project_iam_member" "app_cloudtrace" {
  project = "tasks-510111"
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.app.email}"
}

# Cloud Run
resource "google_project_service" "cloud_run" {
  service = "run.googleapis.com"

  disable_on_destroy = false
}

resource "google_cloud_run_v2_service" "app" {
  depends_on = [
    google_artifact_registry_repository.main,
    google_project_service.cloud_run,
    google_service_account.app,
    google_storage_bucket.db,
  ]

  name     = "tasks-app"
  location = "asia-northeast1"
  ingress  = "INGRESS_TRAFFIC_ALL"

  deletion_protection = true

  template {
    service_account = google_service_account.app.email

    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }

    containers {
      image = "${google_artifact_registry_repository.main.location}-docker.pkg.dev/tasks-510111/${google_artifact_registry_repository.main.name}/app:${local.image_tag}"
      name  = "app"

      ports {
        container_port = 8000
      }

      env {
        name = "ADMIN_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.admin_password.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "ADMIN_USERNAME"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.admin_username.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "DATABASE_URL"
        value = "file:/app/app.db"
      }
      env {
        name  = "GOOGLE_CLOUD_PROJECT_NAME"
        value = "tasks-510111"
      }
      env {
        name  = "OTEL_COLLECTOR_URL"
        value = "localhost:4317"
      }
      env {
        name  = "USE_SECURE_COOKIE"
        value = "true"
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "256Mi"
        }
        cpu_idle = true
      }
    }
    containers {
      image = "${google_artifact_registry_repository.main.location}-docker.pkg.dev/tasks-510111/${google_artifact_registry_repository.main.name}/otelcol:${local.image_tag}"
      name  = "otelcol"

      resources {
        limits = {
          cpu    = "1000m"
          memory = "256Mi"
        }
        cpu_idle = true
      }
    }
  }
}

resource "google_cloud_run_service_iam_policy" "cloud_run_noauth" {
  location    = google_cloud_run_v2_service.app.location
  project     = google_cloud_run_v2_service.app.project
  service     = google_cloud_run_v2_service.app.name
  policy_data = data.google_iam_policy.cloud_run_noauth.policy_data
}

data "google_iam_role" "run_invoker" {
  name = "roles/run.invoker"
}

data "google_iam_policy" "cloud_run_noauth" {
  binding {
    role    = data.google_iam_role.run_invoker.name
    members = ["allUsers"]
  }
}

output "url" {
  value = google_cloud_run_v2_service.app.uri
}

# Custom Domain
data "google_project" "project" {}

resource "google_cloud_run_domain_mapping" "main" {
  depends_on = [
    google_cloud_run_v2_service.app
  ]

  name     = "tasks.fjnkt98.com"
  location = google_cloud_run_v2_service.app.location
  metadata {
    namespace = data.google_project.project.project_id
  }
  spec {
    route_name = google_cloud_run_v2_service.app.name
  }
}
