terraform {
  required_providers {
    openwebui = {
      source  = "chrisguidry/openwebui"
      version = "~> 1.4"
    }
  }
}

provider "openwebui" {
  endpoint = "https://openwebui.example.com"
  token    = var.openwebui_token
}

variable "openwebui_token" {
  type        = string
  sensitive   = true
  description = "Admin API token for the Open WebUI instance."
}

# The same settings come from the environment when the provider block leaves
# them out, which keeps the token out of the configuration entirely:
#
#   export OPENWEBUI_ENDPOINT=https://openwebui.example.com
#   export OPENWEBUI_TOKEN=...
#
# provider "openwebui" {}
