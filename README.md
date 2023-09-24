# Build a *bare metal managed* Kubernetes cluster on Scaleway with power consumption monitoring using Scaphandre

## Setup the cluster (30min)

1. Create the Elastic Metal server with Debian 12 and an Intel CPU

```bash
apt-get update && apt-get upgrade
apt-get install -y ufw
ufw disable
```

2. Create a Kosmos Kubernetes cluster with a external pool
3. Attach the Elasctic Metal server to the external pool (follow documentation)
4. Download the kubeconfig file

## Deploy the observability stack (10min)

1. Create the observability namespace

```bash
kubectl create namespace observability
```

2. Deploy the kube-prometheus-stack helm chart

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack --namespace observability
```

3. Deploy the scaphandre helm chart

```bash
git clone https://github.com/hubblo-org/scaphandre
cd scaphandre
git switch dev
helm install scaphandre helm/scaphandre \
    --set image.name=airklizz/myscaphandre \
    --set image.tag=kuberegexmodified \
    --set serviceMonitor.enabled=true \
    --set serviceMonitor.namespace=observability \
    --set serviceMonitor.interval=30s \
    --namespace observability
```

> Add label `release: kube-prometheus-stack` to the scaphandre service monitor.

4. Deploy online boutique

```bash
helm upgrade onlineboutique oci://us-docker.pkg.dev/online-boutique-ci/charts/onlineboutique \
    --install \
    --create-namespace \
    -n onlineboutique
```

5. Add electricity map exporter

Create the secret with Electricity Map API TOKEN:

```bash
kubectl create secret -n observability generic electricitymap-exporter-secret --from-literal=AUTH_TOKEN=<token>
```

Deploy the exporter:

```bash
kubectl apply -f electricitymap_exporter/kube.yaml
```

6. Prometheus metrics

Nodes consumption for the last 3 hours:

```promql
sum_over_time(
  avg_over_time(
    avg(
      (sum(scaph_host_power_microwatts)) 
      / 1000000 * 1.4 * 
      ignoring(instance, container, endpoint, job, namespace, pod, service) carbon_intensity_fr 
      / 1000
    )[1h:1h]
  )[3h:1h]
)
```


# Expose metrics accros KVM/QEMU

/!\ I had to host my own container image because it doesn't enable --features qemu on actual hubblo/scaphandre:dev compiled version . /!\

1. On bare metal hypervisor, run scaphandre in qemu exporter mode : 
```docker run -d -p 8080:8080 -v /sys/class/powercap/:/sys/class/powercap -v /proc:/proc -v /var/lib/libvirt/scaphandre/:/var/lib/libvirt/scaphandre/ ghcr.io/damienvergnaud/scaphandre-kubernetes/scaphandre:dev2 qemu```

> Scaphandre will split consumptions metrics found in bare-metal host per virtual machine under Kvm/Qemu. 
> But mapping it to expose it under each VM will be your duty

2. For each VM creation do this on bare-metal hypervisor : 
```mount -t tmpfs tmpfs_<DOMAIN_QEMU_VM> /var/lib/libvirt/scaphandre/<DOMAIN_QEMU_VM> -o size=10m```

3. Then on kvm/qemu configuration, you will have to define this mapping per VM
> Scaphandre documentation recommand using virt-manager, it's a good recommandation, use it, otherwise :  
```sudo virsh edit <DOMAIN_NAME>```

> It only worked using virtio-p9 as a driver for me.

```
    <filesystem type='mount' accessmode='mapped'>
      <source dir='/var/lib/libvirt/scaphandre/ubuntuvierge'/>
      <target dir='/dev/scaphandre'/>
    </filesystem>
```

On guest : 
1. Mount the newly created virtio-9p mountpoint in /var/scaphandre
```mount -t 9p -o trans=virtio scaphandre /var/scaphandre

Start Scaphandre in VM mode (Thus, scaphandre is searching for RAPL in /var/scaphandre/ instead of /sys/class/powercap/
```podman run -p 8080:8080 -v /sys/class/powercap/:/sys/class/powercap -v /proc:/proc -it hubblo/scaphandre:dev prometheus --vm -s5```

Check for metrics prometheus metrics :
```curl localhost:8080/metrics" now return metrics
