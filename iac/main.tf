###############################################
#        SETUP THE SCALEWAY PROVIDER          #
###############################################

terraform {
  required_providers {
    scaleway  = {
      source  = "scaleway/scaleway"
      version = "= 2.27.0"
    }
  }
  required_version = "= 1.5.6"
}


###############################################
#         CONFIGURE THE K8S CLUSTER           #
###############################################

resource "scaleway_k8s_cluster" "multicloud" {
  name    = "multicloud-cluster"
  type    = "multicloud"
  version = "1.27.1"
  cni     = "kilo"
  region  = "fr-par"
  delete_additional_resources = false
}

resource "scaleway_k8s_pool" "pool" {
  cluster_id  = scaleway_k8s_cluster.multicloud.id
  name        = "multicloud-pool"
  node_type   = "external"
  size        = 0
  region      = "fr-par"
}

###############################################
#     CONFIGURE THE ELASTIC METAL SERVER      #
###############################################

# Select at least one SSH key to connect to your server
resource "scaleway_iam_ssh_key" "key" {
  name = "hello-remote-node"
  public_key = file(".id_rsa.pub")
}
# Select the type of offer for your server
data "scaleway_baremetal_offer" "offer" {
  name = "EM-B112X-SSD"
}
# Select the OS you want installed on your server
data "scaleway_baremetal_os" "os" {
  name = "Ubuntu"
  version = "20.04 LTS (Focal Fossa)"
}

data "local_sensitive_file" "secret_key" {
    filename = pathexpand(".id_rsa.pkey")
}

resource "scaleway_baremetal_server" "server" {
    offer       = data.scaleway_baremetal_offer.offer.name
    os          = data.scaleway_baremetal_os.os.id
    ssh_key_ids = [scaleway_iam_ssh_key.key.id]

    # Configure the SSH connexion used by Terraform for the remote execution  
    connection {
      type     = "ssh"
      user     = "ubuntu"
      host     = one([for k in self.ips : k if k.version == "IPv4"]).address   # We look for the IPv4 in the list of IPs
    }

    # Download and execute the configuration script
    provisioner "remote-exec" {
      inline = [
        "wget https://scwcontainermulticloud.s3.fr-par.scw.cloud/node-agent_linux_amd64 > log && chmod +x node-agent_linux_amd64",
        "echo \"\nPOOL_ID=${split("/", scaleway_k8s_pool.pool.id)[1]}\nPOOL_REGION=${scaleway_k8s_pool.pool.region}\nSCW_SECRET_KEY=${data.local_sensitive_file.secret_key.content}\" >> log",
        "export POOL_ID=${split("/", scaleway_k8s_pool.pool.id)[1]}  POOL_REGION=${scaleway_k8s_pool.pool.region}  SCW_SECRET_KEY=${data.local_sensitive_file.secret_key.content}",
        "sudo ./node-agent_linux_amd64 -loglevel 0 -no-controller >> log",
      ]
    }
}