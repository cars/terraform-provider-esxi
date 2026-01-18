variable "esxi_hostname" {
  description = "ESXi hostname or IP address"
  type        = string
  default     = "esxi"
}

variable "esxi_hostport" {
  description = "ESXi SSH port"
  type        = string
  default     = "22"
}

variable "esxi_username" {
  description = "ESXi SSH username"
  type        = string
  default     = "root"
}

variable "esxi_password" {
  description = "ESXi SSH password"
  type        = string
  sensitive   = true
}
