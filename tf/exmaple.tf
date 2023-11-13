resource "google_compute_address" "bastion" {
  name = "test-bastion"
}

resource "google_dns_record_set" "bastion" {
  managed_zone = "private-test-zone"
  name         = "bastion.test.com"
  type         = "A"
  rrdatas      = [google_compute_address.bastion.address]
  ttl          = 3600
}

resource "google_compute_firewall" "bastion" {
  name          = "test-bastion"
  network       = "test-network"
  target_tags   = ["bastion"]
  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]
  allow {
    protocol = "tcp"
    ports    = [22]
  }
}

data "google_compute_image" "bastion" {
  project = "arch-linux-gce"
  name    = "arch-v20210124"
}

resource "google_compute_disk" "bastion" {
  name   = "test-bastion"
  type   = "pd-standard"
  image  = data.google_compute_image.bastion.id
  labels = [{ workspace : "compute" }]
}

resource "google_compute_instance" "bastion" {
  name         = "test-bastion"
  machine_type = "e2-micro"
  boot_disk {
    source      = google_compute_disk.bastion.name
    auto_delete = false
  }
  network_interface {
    subnetwork = "test-subnetwork"
    access_config {
      nat_ip = google_compute_address.bastion.address
    }
  }
  tags = ["bastion"]
  service_account {
    email = var.service_account
    scopes = ["https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
    "https://www.googleapis.com/auth/trace.append"]
  }

  allow_stopping_for_update = true
  scheduling {
    automatic_restart = true
  }
}
