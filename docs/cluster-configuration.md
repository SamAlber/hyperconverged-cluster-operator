# Cluster Configuration
## Introduction
The HyperConverged Cluster allows modifying the KubeVirt cluster configuration by editing the HyperConverged Cluster CR
(Custom Resource).

The HyperConverged Cluster operator copies the cluster configuration values to the other operand's CRs.

The Hyperconverged Cluster Operator configures kubevirt and its supporting operators in an opinionated way and overwrites its operands when there is an unexpected change to them.
Users are expected to not modify the operands directly. The HyperConverged custom resource is the source of truth for the configuration.

To make it more visible and clear for end users, the Hyperconverged Cluster Operator will count the number of these revert actions in a metric named kubevirt_hco_out_of_band_modifications_total.
According to the value of that metric in the last 10 minutes, an alert named KubeVirtCRModified will be eventually fired:
```
Labels
    alertname=KubeVirtCRModified
    component_name=kubevirt-kubevirt-hyperconverged
    severity=warning
```
The alert is supposed to resolve after 10 minutes if there isn't a manual intervention to operands in the last 10 minutes.

***Note***: The cluster configurations are supported only in API version `v1beta1` or higher.
## Infra and Workloads Configuration
Some configurations are done separately to Infra and Workloads. The CR's Spec object contains the `infra` and the
`workloads` objects.

The structures of the `infra` and the `workloads` objects are the same. The HyperConverged Cluster operator will update
the other operator CRs, according to the specific CR structure. The meaning is if, for example, the other CR does not
support Infra cluster configurations, but only Workloads configurations, the HyperConverged Cluster operator will only
copy the Workloads configurations to this operator's CR.

Below are the cluster configuration details. Currently, only "Node Placement" configuration is supported.

### Node Placement
Kubernetes lets the cluster admin influence node placement in several ways, see
https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for a general overview.

The HyperConverged Cluster's CR is the single entry point to let the cluster admin influence the placement of all the pods directly and indirectly managed by the HyperConverged Cluster Operator.

The `nodePlacement` object is an optional field in the HyperConverged Cluster's CR, under `spec.infra` and `spec.workloads`
fields.

***Note***: The HyperConverged Cluster operator does not allow modifying of the workloads' node placement configurations if there are already
existing virtual machines or data volumes.

The `nodePlacement` object contains the following fields:
* `nodeSelector` is the node selector applied to the relevant kind of pods. It specifies a map of key-value pairs: for
the pod to be eligible to run on a node,	the node must have each of the indicated key-value pairs as labels 	(it can
have additional labels as well). See https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector.
* `affinity` enables pod affinity/anti-affinity placement expanding the types of constraints
that can be expressed with nodeSelector.
affinity is going to be applied to the relevant kind of pods in parallel with nodeSelector
See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity.
* `tolerations` is a list of tolerations applied to the relevant kind of pods.
See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more info.

#### Operators placement
The HyperConverged Cluster Operator and the operators for its component are supposed to be deployed by the Operator Lifecycle Manager (OLM).
Thus, the HyperConverged Cluster Operator is not going to directly influence its own placement but that should be influenced by the OLM.
The cluster admin indeed is allowed to influence the placement of the Pods directly created by the OLM configuring a [nodeSelector](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/subscription-config.md#nodeselector) or [tolerations](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/subscription-config.md#tolerations) directly on the OLM subscription object.

#### Node Placement Examples
* Place the infra resources on nodes labeled with "nodeType = infra", and workloads in nodes labeled with "nodeType = nested-virtualization", using node selector:
  ```yaml
  ...
  spec:
    infra:
      nodePlacement:
        nodeSelector:
          nodeType: infra
    workloads:
      nodePlacement:
        nodeSelector:
          nodeType: nested-virtualization
  ```
* Place the infra resources on nodes labeled with "nodeType = infra", and workloads in nodes labeled with
"nodeType = nested-virtualization", preferring nodes with more than 8 CPUs, using affinity:
  ```yaml
  ...
  spec:
    infra:
      nodePlacement:
        affinity:
          nodeAffinity:
            requiredDuringSchedulingIgnoredDuringExecution:
              nodeSelectorTerms:
              - matchExpressions:
                - key: nodeType
                  operator: In
                  values:
                  - infra
    workloads:
      nodePlacement:
        affinity:
          nodeAffinity:
            requiredDuringSchedulingIgnoredDuringExecution:
              nodeSelectorTerms:
              - matchExpressions:
                - key: nodeType
                  operator: In
                  values:
                  - nested-virtualization
            preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 1
              preference:
                matchExpressions:
                - key: my-cloud.io/num-cpus
                  operator: gt
                  values:
                  - 8
  ```
* In this example, there are several nodes that are saved for KubeVirt resources (e.g. VMs), already set with the
`key=kubevirt:NoSchedule` taint. This taint will prevent any scheduling to these nodes, except for pods with the matching
tolerations.
  ```yaml
  ...
  spec:
    workloads:
      nodePlacement:
        tolerations:
        - key: "key"
          operator: "Equal"
          value: "kubevirt"
          effect: "NoSchedule"
  ```

## FeatureGates
The `featureGates` field is an optional set of optional boolean feature enabler. The features in this list are advanced
or new features that are not enabled by default.

To enable a feature, add its name to the `featureGates` list and set it to `true`. Missing or `false` feature gates
disables the feature.

### downwardMetrics Feature Gate
Set the `downwardMetrics` feature gate in order to allow exposing a limited set of VM and host metrics to the guest.
The format is compatible with [vhostmd](https://github.com/vhostmd/vhostmd).
These metrics allow third-parties diagnosing issues.
DownwardMetrics may be exposed to the guest through a `volume` or a `virtio-serial port`.

**Note**: Updates from previous versions requires enabling this feature gate if the metrics were used in any
virtual machine.

**Note**: this feature is in Developer Preview.

**Default**: `false`

**Graduation Status**: Alpha

## withHostPassthroughCPU Feature Gate
This feature gate is deprecated and is ignored.

### deployKubeSecondaryDNS Feature Gate
Set the `deployKubeSecondaryDNS` feature gate to true to allow deploying KubeSecondaryDNS by CNAO.
For additional information, see here: [KubeSecondaryDNS](https://github.com/kubevirt/kubesecondarydns)

**Default**: `false`

### persistentReservation Feature Gate
Set the `persistentReservation` feature gate to true in order to enable the reservation of a LUN through the SCSI Persistent Reserve commands.

SCSI protocol offers dedicated commands in order to reserve and control access to the LUNs. This can be used to prevent data corruption if the disk is shared by multiple VMs (or more in general processes).
The SCSI persistent reservation is handled by the qemu-pr-helper. The pr-helper is a privileged daemon that can be either started by libvirt directly or managed externally.
In case of KubeVirt, the qemu-pr-helper needs to be started externally because it requires high privileges in order to perform the persistent SCSI reservation. Afterward, the pr-helper socket is accessed by the unprivileged virt-launcher pod for enabling the SCSI persistent reservation.
Once the feature gate is enabled, then the additional container with the qemu-pr-helper is deployed inside the virt-handler pod. Enabling (or removing) the feature gate causes the redeployment of the virt-handler pod.

VMI example:
```yaml
    devices:
      disks:
      - name: mypvcdisk
        lun:
          reservations: true
```
**Note**: An important aspect of this feature is that the SCSI persistent reservation doesn't support migration. Even if you apply the reservation to an RWX PVC provisioning SCSI devices, the restriction is due to the reservation done by the initiator on the node. The VM could be migrated but not the reservation.

**Note**: this feature is in Developer Preview.

**Default**: `false`

**Graduation Status**: Alpha

### alignCPUs Feature Gate
Set the `alignCPUs` feature gate to enable KubeVirt
to request up to two additional dedicated CPUs in order to complete the total CPU count
to an even parity when using emulator thread isolation.

**Note**: this feature is in Developer Preview.

**Default**: `false`

**Graduation Status**: Alpha

### disableMDevConfiguration Feature Gate
KubeVirt aims to facilitate the configuration of mediated devices on large clusters.

If this is not desired, set the `disableMDevConfiguration` feature gate in order to disable this feature.

**Note**: this feature is in Developer Preview.

**Default**: `false`

**Graduation Status**: Alpha

### decentralizedLiveMigration Feature Gate
By default, live migration is limited in its flexibility because the migration is centralized. This limits live
migration to the same namespace and cluster. There are use cases where it would be beneficial to live migrate between
namespaces and even between clusters. For instance for balancing VirtualMachines between clusters or promoting a VM from
a UAT namespace to the production namespace.

Set the `decentralizedLiveMigration` feature gate to true in order to enable decentralized live migration.

**Note**: this feature is in Developer Preview.

**Default**: `false`

**Graduation Status**: Alpha

### The hco.kubevirt.io/deployPasstNetworkBinding annotation
Set the `hco.kubevirt.io/deployPasstNetworkBinding` HyperConverged CR annotation to `true`, to deploy the needed
configurations for kubevirt users, so they can bind their VM using a Passt Network binding.

**Note**: this feature is in Developer Preview.

**Default**: `false` (annotation doesn't exist by default)

**Graduation Status**: Alpha

### Feature Gates Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  infra: {}
  workloads: {}
  featureGates:
    alignCPUs: true
    deployKubeSecondaryDNS: true
```

## Live Migration Configurations

Set the live migration configurations by modifying the fields in the `liveMigrationConfig` under the `spec` field

### bandwidthPerMigration

Bandwidth limit of each migrationi, the value is in quantity of bytes per second. The format is a number with an optional quantity suffix, e.g. `2048Mi` means 2048MiB/sec.

**default**: unset

### completionTimeoutPerGiB

If a migrating VM is big and busy, while the connection to the destination node 
is slow, migration may never converge. The completion timeout is calculated
based on completionTimeoutPerGiB times the size of the guest (both RAM and 
migrated disks, if any). For example, with completionTimeoutPerGiB set to 800,
a virtual machine instance with 6GiB memory will timeout if it has not
completed migration in 1h20m. Use a lower completionTimeoutPerGiB to induce
quicker failure, so that another destination or post-copy is attempted. Use a
higher completionTimeoutPerGiB to let workload with spikes in its memory dirty
rate to converge.
The format is a number.

**default**: 800

### parallelMigrationsPerCluster

Number of migrations running in parallel in the cluster. The format is a number.

**default**: 5

### parallelOutboundMigrationsPerNode

Maximum number of outbound migrations per node. The format is a number.

**default**: 2

### progressTimeout:

The migration will be canceled if memory copy fails to make progress in this time, in seconds. The format is a number.

**default**: 150

### network

The name of a [Multus](https://github.com/k8snetworkplumbingwg/multus-cni) network attachment definition to be dedicated to live migrations to minimize disruption to tenant workloads due to network saturation when VM live migrations are triggered. The format is a string.

**default**: unset

### allowAutoConverge

It allows the platform to compromise performance/availability of VMIs to guarantee successful VMI live migrations.

**default**: false

### allowPostCopy

When enabled, KubeVirt attempts to use post-copy live-migration in case it 
reaches its completion timeout while attempting pre-copy live-migration.
Post-copy migrations allow even the busiest VMs to successfully live-migrate.
However, events like a network failure or a failure in any of the source or
destination nodes can cause the migrated VM to crash.
Enable this option when evicting nodes is more important than keeping VMs
alive.

**default**: false

### Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  liveMigrationConfig:
    completionTimeoutPerGiB: 800
    network: migration-network
    parallelMigrationsPerCluster: 5
    parallelOutboundMigrationsPerNode: 2
    progressTimeout: 150
    allowAutoConverge: false
    allowPostCopy: false
```

## Automatic Configuration of Mediated Devices (including vGPUs)

Administrators can provide a list of desired mediated devices (vGPU) types.
KubeVirt will attempt to automatically create the relevant devices on nodes that can support such configuration.
Currently, it is possible to configure one type per physical card.
KubeVirt will configure all `available_instances` for each configurable type.

### Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  mediatedDevicesConfiguration:
    mediatedDeviceTypes:
      - nvidia-222
      - nvidia-228
      - i915-GVTg_V5_4
```

Administrators are able to expand the mediatedDevicesConfiguration API to allow a more specific per-node configuration, using
NodeSelectors to target a sepcific node.


### Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  mediatedDevicesConfiguration:
    mediatedDeviceTypes:
      - nvidia-222
      - nvidia-228
      - i915-GVTg_V5_4
    nodeMediatedDeviceTypes:
      - nodeSelector:
          someLabel1: ""
        mediatedDeviceTypes:
        - nvidia-222
      - nodeSelector:
          kubernetes.io/hostname: nodeName
        mediatedDeviceTypes:
        - nvidia-228
```

This API will facilitate the creation of mediated devices types on cluster
nodes. However, administrators are expected to use the PermittedHostDevices API
to allow these devices in the cluster.



## Listing Permitted Host Devices
Administrators can control which host devices are exposed and permitted to be used in the cluster. Permitted host
devices in the cluster will need to be allowlisted in KubeVirt CR by its `vendor:product` selector for PCI devices or
mediated device names. Use the `permittedHostDevices` field in order to manage the permitted host devices.

The `permittedHostDevices` field is an optional field under the HyperConverged `spec` field.

The `permittedHostDevices` field contains three optional arrays: the `pciHostDevices`, `mediatedDevices` and `usbHostDevices` array.

HCO propagates these arrays as is to the KubeVirt custom resource; i.e. no merge is done, but a replacement.

The `pciHostDevices` array is an array of `PciHostDevice` objects. The fields of this object are:
* `pciDeviceSelector` - a combination of a **`vendor_id:product_id`** required to identify a PCI device on a host.

   This identifier 10de:1eb8 can be found using `lspci`; for example:
   ```shell
   lspci -nnv | grep -i nvidia
   ```
  
* `resourceName` - name by which a device is advertised and being requested.
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.

  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and
  monitoring to an external device plugin.

  **default**: `false`

The `mediatedDevices` array is an array of `MediatedDevice` objects. The fields of this object are:
* `mdevNameSelector` - name of a mediated device type required to identify a mediated device on a host.

   For example: mdev type nvidia-226 represents GRID T4-2A.

  The selector is matched against the content of `/sys/class/mdev_bus/$mdevUUID/mdev_type/name`.
* `resourceName` - name by which a device is advertised and being requested.
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.

  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and
  monitoring to an external device plugin.

  **default**: `false`

The `usbHostDevices` array is an array of `USBHostDevice` objects. The fields of this object are:
* `selectors` array is an array of `USBSelector` objects. The fields of this object are a **`vendor`, `product`** required to identify a USB device.

  The two identifiers like `06cb` and `00f9` can be found using `lsusb`; for example:
   ```shell
   $ lsusb | grep -i synaptics
   Bus 003 Device 005: ID 06cb:00f9 Synaptics, Inc.
   ```
  Multiple `USBSelector` objects can be used to match a st of devices.

* `resourceName` - name by which a device is advertised and being requested.
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.

  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and
  monitoring to an external device plugin.

  **default**: `false`

### Permitted Host Devices Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  permittedHostDevices:
    pciHostDevices:
    - pciDeviceSelector: "10DE:1DB6"
      resourceName: "nvidia.com/GV100GL_Tesla_V100"
    - pciDeviceSelector: "10DE:1EB8"
      resourceName: "nvidia.com/TU104GL_Tesla_T4"
    mediatedDevices:
    - mdevNameSelector: "GRID T4-1Q"
      resourceName: "nvidia.com/GRID_T4-1Q"
    usbHostDevices:
    - resourceName: kubevirt.io/peripherals
      selectors:
        - vendor: "045e"
          product: "07a5"
        - vendor: "062a"
          product: "4102"
        - vendor: "072f"
          product: "b100"
      externalResourceProvider: false
    - resourceName: kubevirt.io/storage
      selectors:
        - vendor: "46f4"
          product: "0001"
      externalResourceProvider: false
```

## Filesystem Overhead

By default, when using DataVolumes with storage profiles (spec.storage is non-empty), the size of a Filesystem
PVC chosen is bigger, ensuring 5.5% of the space isn't used. This is to account for root reserved blocks as
well as to avoid reaching full capacity, as file systems often have degraded performance and perhaps weak
guarantees about the amount of space that can be fully occupied.

In case a larger or smaller padding is needed, for example if the root reservation is larger or the file
system recommends a larger percentage of the space should be unused for optimal performance, we can change
this tunable through the HCO CR.  
This is possible to do as a global setting and per-storage class name.

Administrators can Override the storage class used for scratch space during transfer operations by setting the
`scratchSpaceStorageClass` field under the HyperConverged `spec` field.

The scratch space storage class is determined in the following order:

value of scratchSpaceStorageClass, if that doesn't exist, use the default storage class, if there is no default storage
class, use the storage class of the DataVolume, if no storage class specified, use no storage class for scratch space

### Storage Class for Scratch Space Example

For example, if we want the 'nfs' storage class to not use any padding, and the global setting to be 10% instead,
we can add the following parts to the HCO spec:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  filesystemOverhead:
    global: "0.1"
    storageClass:
      nfs: "0"
```


## Storage Class for Scratch Space

Administrators can Override the storage class used for scratch space during transfer operations by setting the
`scratchSpaceStorageClass` field under the HyperConverged `spec` field.

The scratch space storage class is determined in the following order:

value of scratchSpaceStorageClass, if that doesn't exist, use the default storage class, if there is no default storage
class, use the storage class of the DataVolume, if no storage class specified, use no storage class for scratch space

### Storage Class for Scratch Space Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  scratchSpaceStorageClass: aStorageClassName
```

## Resource Requests

### VMI PODs CPU Allocation Ratio

KubeVirt runs Virtual Machines in a Kubernetes Pod.
This pod requests a certain amount of CPU time from the host.
On the other hand, the Virtual Machine is being created with a certain amount of vCPUs.
The number of vCPUs may not necessarily correlate to the number of requested CPUs by the POD.
Depending on the QOS of the POD, vCPUs can be scheduled on a variable amount of physical CPUs; this depends on the available CPU resources on a node.
When there are fewer available CPUs on the node as the requested vCPU, vCPU will be over committed.
By default, each pod requests 100mil of CPU time. The CPU requested on the pod sets the cgroups cpu.shares which serves as a priority for the scheduler to provide CPU time for vCPUs in this POD.
As the number of vCPUs increases, this will reduce the amount of CPU time each vCPU may get when competing with other processes on the node or other Virtual Machine Instances with a lower amount of vCPUs.
The vmiCPUAllocationRatio comes to normalize the amount of CPU time the POD will request based on the number of vCPUs.
POD CPU request = number of vCPUs * 1/cpuAllocationRatio
For example, a value of 1 means 1 physical CPU thread per VMI CPU thread.
A value of 100 would be 1% of a physical thread allocated for each requested VMI thread.
The default value is 10.
This option has no effect on VMIs that request dedicated CPUs.

**Note**: In Kubernetes, one full core is 1000 of CPU time More Information

Administrators can change this ratio by updating the HCO CR.

#### VMI PODs CPU request example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  resourceRequirements:
    vmiCPUAllocationRatio: 16
```

### Storage Resource Configurations

The administrator can limit storage workloads resources and to require minimal resources. Use the `resourceRequirements`
field under the HyperConverged `spec` filed. Add the `storageWorkloads` field under the `resourceRequirements`. The
content of the `storageWorkloads` field is
the [standard kubernetes resource configuration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#resourcerequirements-v1-core)
.

#### Storage Resource Configurations Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  resourceRequirements:
    storageWorkloads:
      limits:
        cpu: "500m"
        memory: "2Gi"
      requests:
        cpu: "250m"
        memory: "1Gi"
```

## Cert Rotation Configuration
You can configure certificate rotation parameters to influence the frequency of the rotation of the certificates needed by a Kubevirt deployment.

All values must be 10 minutes in length or greater to avoid overloading the system, and should be expressed as strings that comply with [golang's ParseDuration format](https://golang.org/pkg/time/#ParseDuration).
The value of `server.duration` should be less than the value of `ca.duration`.
The value of `renewBefore` should be less than the value of `server.duration`.

### Cert Rotation Configuration Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
  namespace: kubevirt-hyperconverged
spec:
  certConfig:
    ca:
      duration: 48h0m0s
      renewBefore: 24h0m0s
    server:
      duration: 24h0m0s
      renewBefore: 12h0m0s
```

## CPU Plugin Configurations
You can schedule a virtual machine (VM) on a node where the CPU model and policy attribute of the VM are compatible with
the CPU models and policy attributes that the node supports. By specifying a list of obsolete CPU models in the
HyperConverged custom resource, you can exclude them from the list of labels created for CPU models.

Through the process of iteration, the list of base CPU features in the minimum CPU model are eliminated from the list of
labels generated for the node. For example, an environment might have two supported CPU models: `Penryn` and `Haswell`.

Use the `spec.obsoleteCPUs` to configure the CPU plugin. Add the obsolete CPU list under `spec.obsoleteCPUs.cpuModels`,
and the minCPUModel as the value of `spec.obsoleteCPUs.minCPUModel`.

The default value for the `spec.obsoleteCPUs.minCPUModel` field in KubeVirt is `"Penryn"`, but it won't be visible if
missing in the CR. The default value for the `spec.obsoleteCPUs.cpuModels` field is hardcoded predefined list and is not
visible. You can add new CPU models to the list, but can't remove CPU models from the predefined list. The predefined list
is not visible in the HyperConverged CR.

The hard-coded predefined list of obsolete CPU modes is:
* `486`
* `pentium`
* `pentium2`
* `pentium3`
* `pentiumpro`
* `coreduo`
* `n270`
* `core2duo`
* `Conroe`
* `athlon`
* `phenom`
* `qemu64`
* `qemu32`
* `kvm64`
* `kvm32`

You don't need to add a CPU model to the `spec.obsoleteCPUs.cpuModels` field if it is in this list.

### CPU Plugin Configurations Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  obsoleteCPUs:
    cpuModels:
      - "486"
      - "pentium"
      - "pentium2"
      - "pentium3"
      - "pentiumpro"
    minCPUModel: "Penryn"
```

## Default CPU model configuration
User can specify a cluster-wide default CPU model: default CPU model is set when vmi doesn't have any cpu model.
When vmi has cpu model set, then vmi's cpu model is preferred. When default cpu model is not set and vmi's cpu model is not set too, host-model will be set.
Default cpu model can be changed when kubevirt is running.
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  defaultCPUModel: "EPYC"
```

## Default RuntimeClass
User can specify a cluster-wide default RuntimeClass for VMIs pods: default RuntimeClass is set when vmi doesn't have any specific RuntimeClass.
When vmi RuntimeClass is set, then vmi's RuntimeClass is preferred. When default RuntimeClass is not set and vmi's RuntimeClass is not set too, RuntimeClass will not be configured on VMIs pods .
Default RuntimeClass can be changed when kubevirt is running, existing VMIs are not impacted till the next restart/live-migration when they are eventually going to consume the new default RuntimeClass.
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  defaultRuntimeClass: "myCustomRuntimeClass"
```

## Common templates namespace
User can specify namespace in which common templates will be deployed. This will override default `openshift` namespace.
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  commonTemplatesNamespace: kubevirt
```

## Tekton Pipelines namespace
User can specify namespace in which example pipelines will be deployed.
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  tektonPipelinesNamespace: kubevirt
```
In case the namespace is unspecified, the operator namespace will serve as the default value.

## Tekton Tasks namespace
User can specify namespace in which tekton tasks will be deployed.
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  tektonTasksNamespace: kubevirt
```
In case the namespace is unspecified, the operator namespace will serve as the default value.

## Enable eventual launcher updates by default
us the HyperConverged `spec.workloadUpdateStrategy` object to define how to handle automated workload updates at the cluster
level.

The `workloadUpdateStrategy` fields are:
* `batchEvictionInterval` - BatchEvictionInterval Represents the interval to wait before issuing the next batch of
  shutdowns.

  The Default value is `1m`
  
* `batchEvictionSize` - Represents the number of VMIs that can be forced updated per the BatchShutdownInterval interval

  The default value is `10`

* `workloadUpdateMethods` - defines the methods that can be used to disrupt workloads
  during automated workload updates.
  
  When multiple methods are present, the least disruptive method takes
  precedence over more disruptive methods. For example if both LiveMigrate and Shutdown
  methods are listed, only VMs which are not live migratable will be restarted/shutdown.
  
  An empty list defaults to no automated workload updating.

  The default values is `LiveMigrate`; `Evict` is not enabled by default being potentially disruptive for the existing workloads.

### workloadUpdateStrategy example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  workloadUpdateStrategy:
    workloadUpdateMethods:
    - LiveMigrate
    - Evict
    batchEvictionSize: 10
    batchEvictionInterval: "1m"
```

## Insecure Registries for Imported Data containerized Images
If there is a need to import data images from an insecure registry, these registries should be added to the
`insecureRegistries` field under the `storageImport` in the `HyperConverged`'s `spec` field.

### Insecure Registry Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  storageImport:
    insecureRegistries:
      - "private-registry-example-1:5000"
      - "private-registry-example-2:5000"
```

## KubeSecondaryDNS Name Server IP
Set the `spec.deployKubeSecondaryDNS` field gate to `true` to allow deploying KubeSecondaryDNS by CNAO.

In order to set KSD's NameServerIP, set it on HyperConverged CR under spec.kubeSecondaryDNSNameServerIP field.
Default: empty string. Value is a string representation of IPv4 (i.e "127.0.0.1").
For more info see [deployKubeSecondaryDNS Feature Gate](#deploykubesecondarydns-feature-gate).

### KubeSecondaryDNS Name Server IP example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  kubeSecondaryDNSNameServerIP: "127.0.0.1"
```

## Network Binding plugin
In order to set NetworkBinding, set it on HyperConverged CR under spec.networkBinding field.
Default: empty map (no binding plugins).

### Network Binding plugin example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  networkBinding:
    custom-binding1:
      sidecarImage: quay.io/custom-binding1-image
    custom-binding2:
      sidecarImage: quay.io/custom-binding2-image
      networkAttachmentDefinition: customBinding2Nad
```

## Modify common golden images
Golden images are root disk images for commonly used operating systems. HCO provides several common images, but it is possible to modify them, if needed.

The list of all the golden images is available at the `status` field, under `dataImportCronTemplates` list. The common images are not part of list in the `spec` field. The list in the status is a reference for modifications. Add the image that requires modification to the list in the spec, and edit it.

The supported modifications are: disabling a specific image, and changing the `storage` field. Editing other fields will be ignored by HCO.

### Disabling all common golden images

Set the `spec.enableCommonBootImageImport` to `false` in order to disable the common golden images in the cluster
(for instance to reduce logs noise on disconnected environments).
For additional information, see
here: https://github.com/kubevirt/community/blob/master/design-proposals/golden-image-delivery-and-update-pipeline.md

**Note**: `spec.enableCommonBootImageImport` is `true` by default.

Example:
```yaml
spec:
  enableCommonBootImageImport: false
```

### Disabling a common golden image
To disable a golden image, add it to the  dataImportCronTemplates` field in the spec object, with the `dataimportcrontemplate.kubevirt.io/enable` annotation, with the value of `false`; for example, disabling the fedora golden image:
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  dataImportCronTemplates:
  - metadata:
      name: fedora-image-cron
      annotations:
        dataimportcrontemplate.kubevirt.io/enable: 'false'
```

There is no need to copy the whole object, but only the relevant fields; i.e. the `metadat.name` field.

### Modifying a common dataImportCronTemplate
It is possible to change the configuration of a common golden image by adding the common image to the
`dataImportCronTemplates` list in the `spec` field. HCO will replace the existing spec object; The `schedule`
field is mandatory, and HCO will copy it from the common template if it is missing.

Copy the required dtaImportCronTemplate object from the list in the `status` field (not including the `status`
field), and change or add the desired fields.

for example, set the storage class for centos8 golden image, and modify the source URL:

```yaml
- metadata:
    name: kubevirt-hyperconverged
  spec:
    schedule: "0 */12 * * *"
    template:
      spec:
        source:
          registry:
            url: docker://my-private-registry/my-own-version-of-centos:8
        storage:
          resources:
            requests:
              storage: 10Gi
            storageClassName: "some-non-default-storage-class"
    garbageCollect: Outdated
    managedDataSource: centos-stream8
```

### Override the golden images namespace
To use another namespace for the common golden images, e.g. in order to hide them, set the `spec.commonBootImageNamespace`
field with the required namespace name.

The customized namespace is ignored for modified golden images.

```yaml
- metadata:
    name: kubevirt-hyperconverged
  spec:
    commonBootImageNamespace: custom-namespace-name
```

## Configure custom golden images
Golden images are root disk images for commonly used operating systems. HCO provides several common images, but it
is also possible to add custom golden images. For more details, see [the golden image documentation](https://github.com/kubevirt/community/blob/master/design-proposals/golden-image-delivery-and-update-pipeline.md).

To add a custom image, add a `DataImportCronTemplate` object to the `dataImportCronTemplates` under
the `HyperConverged`'s
`spec` field.

See the [CDI document](https://github.com/kubevirt/containerized-data-importer/blob/main/doc/os-image-poll-and-update.md) for specific details about the `DataImportCronTemplate` fields.

**Note**: the `enableCommonBootImageImport` feature does not block the custom golden images, but only the common ones.

### Custom Golden Images example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  dataImportCronTemplates:
  - metadata:
      name: custom-image1
    spec:
      schedule: "0 */12 * * *"
      template:
        spec:
          source:
            registry:
              url: docker://myprivateregistry/custom1
      managedDataSource: custom1
      retentionPolicy: "All" # created DataVolumes and DataSources are retained when their DataImportCron is deleted (default behavior)
  - metadata:
      name: custom-image2
    spec:
      schedule: "1 */12 * * *"
      template:
        spec:
          source:
            registry:
              url: docker://myprivateregistry/custom2
      managedDataSource: custom2
      retentionPolicy: "None" # created DataVolumes and DataSources are deleted when their DataImportCron is deleted
```

## Log verbosity
Currently, logging verbosity is only supported for Kubevirt.

### Kubevirt
In order to define logging verbosity for Kubevirt, it's possible to define per-component (e.g. `virt-handler`,
`virt-launcher`, etc) value, or per-node value. All the log verbosity definitions are optional and would automatically
set to a default value by Kubevirt if not value is defined. While there is no tight definition on the behavior for each
component and log verbosity value, the higher the log verbosity value is the higher the verbosity will get.

For example, the following can be configured on HyperConverged CR:
```yaml
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  logVerbosityConfig:
    kubevirt:
      virtLauncher: 8
      virtHandler: 4
      virtController: 1
      nodeVerbosity:
        node01: 4
        node02: 3
```

All the values defined [here](https://kubevirt.io/api-reference/master/definitions.html#_v1_logverbosity)
can be applied.

### Containerized data importer (CDI)
Different levels of verbosity are used in CDI to control the amount of information that is logged. The verbosity level of logs in CDI can be adjusted using a dedicated field that affect all of its components. 
This feature enables users to control the amount of detail displayed in logs, ranging from minimal to detailed debugging information.

For example:
```yaml
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  logVerbosityConfig:
    cdi: 3
```

The verbosity levels in CDI typically range from 1 to 3. Level 1 equates to essential log information, while level 3 delves into more detailed logging, providing more specific information.

## Workloads protection on uninstall

`UninstallStrategy` defines how to proceed on uninstall when workloads (VirtualMachines, DataVolumes) still exist:
- `BlockUninstallIfWorkloadsExist` will prevent the CR from being removed when workloads still exist.
BlockUninstallIfWorkloadsExist is the safest choice to protect your workloads from accidental data loss, so it's strongly advised.
- `RemoveWorkloads` will cause all the workloads to be cascading deleted on uninstallation.
**WARNING**: please notice that RemoveWorkloads will cause your workloads to be deleted as soon as this CR will be, even accidentally, deleted.
Please correctly consider the implications of this option before setting it.

`BlockUninstallIfWorkloadsExist` is the default behaviour.


## Cluster-level eviction strategy

`evictionStrategy` defines at the cluster level if VirtualMachineInstances should be
migrated instead of shut-off in case of a node drain. If the VirtualMachineInstance specific
field is set it overrides the cluster-level one.
Possible values:

- `None` no eviction strategy at cluster level.
- `LiveMigrate` migrate the VM on eviction; a non-live-migratable VM with no specific strategy will block the drain of the node until manually evicted.
- `LiveMigrateIfPossible` migrate the VM on eviction if live migration is possible, otherwise directly evict.
- `External` block the drain, track the eviction and notify an external controller.

`LiveMigrate` is the default behaviour with multiple worker nodes, `None` on single worker clusters.

For example:
```yaml
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  evictionStrategy: LiveMigrateIfPossible
```

## VM state storage class

`VMStateStorageClass` defines the [Kubernetes Storage Class](https://kubernetes.io/docs/concepts/storage/storage-classes/)
to be used for creating persistent state PVCs for VMs, used for example for persisting the state of the vTPM.
The value for this option is the desired storage class name. Example:
```yaml
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  vmStateStorageClass: "rook-cephfs"
```

## Auto CPU limits

`autoCPULimitNamespaceLabelSelector` allows defining a namespace label for which VM pods (virt-launcher) will have a
CPU resource limit of 1 per vCPU.  
This option allows defining namespace CPU quotas equal to the maximum total number of vCPU allowed in that namespace.  
Example:
```yaml
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  resourceRequirements:
    autoCPULimitNamespaceLabelSelector:
      matchLabels:
        autocpulimit: "true"
```
In the example above, VM pods in namespaces that have the label "autocpulimit" set to "true" will have a CPU resource
limit of 1 per vCPU.  
**Important note**: this setting is incompatible with a `vmiCPUAllocationRatio` of 1, since that configuration can lead to
VM pods using more than 1 CPU per vCPU.

## Virtual machine options

`VirtualMachineOptions` holds the cluster level information regarding the virtual machine.
This defines the default behavior of some features related to the virtual machines.


- `DisableFreePageReporting`
  With freePageReporting the guest OS informs the hypervisor about pages which are not
in use by the guest anymore. The hypervisor can use this information for freeing these pages.

  freePageReporting is an attribute that can be defined at [Memory balloon device](https://libvirt.org/formatdomain.html#memory-balloon-device) 
in libvirt. freePageReporting will NOT be enabled for the vmis which does not have the Memballoon driver,
OR which are requesting any high performance components. A vmi is considered as high performance if one of the following is true:
  - the vmi requests a dedicated cpu.
  - the realtime flag is enabled.
  - the vmi requests hugepages.
  
  With `DisableFreePageReporting` freePageReporting will never be enabled in any vmi.
`DisableFreePageReporting` is a boolean and freePageReporting is enabled by default (disableFreePageReporting=false).

- `disableSerialConsoleLog`
  DisableSerialConsoleLog disables logging the auto-attached default serial console.
  If not set, serial console logs will be written to a file in the virt-launcher pod and then streamed from a container named `guest-console-log` so that they can be consumed or aggregated according to the standard k8s logging architecture.
  VM logs streamed over the serial console contain really useful debug/troubleshooting info.  
  The value can be individually overridden for each VM, not relevant if AutoattachSerialConsole is disabled for the VM.

> Caution
> 
> If the serial console is used for interactive access, its output will also be streamed to the log system.
> Passwords are normally not echoed on the screen so they are not going to be collected on the logs.
> However, if you type by mistake a password for a command or if you have to pass it as a clear text parameter it will be written in the logs along with all other visible text.
> If you are inputting any data or commands that contain sensisitive information please consider using SSH unless the serial console is absolutely necessary for troubleshooting.

  `disableSerialConsoleLog` is a boolean, logging of serial console is disabled by default (disableSerialConsoleLog=false).

Example
```yaml
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  virtualMachineOptions:
    disableFreePageReporting: false
    disableSerialConsoleLog: true
```

## KSM Configuration

`ksmConfiguration` instructs on which nodes KSM will be enabled, exposing a `nodeLabelSelector`.  
`nodeLabelSelector` is a [LabelSelector](https://github.com/kubernetes/apimachinery/blob/60180f072f73eafec72ef9f2c418a6bb1357d434/pkg/apis/meta/v1/types.go#L1195)
and defines the filter, based on the node labels. If a node's labels match the label selector term,
then on that node, KSM will be enabled.
>**NOTE**  
>If `nodeLabelSelector` is nil KSM will not be enabled on any nodes.  
>Empty `nodeLabelSelector` will enable KSM on every node.

Examples
- Enabling KSM on nodes in which the hostname is `node01` or `node03`:
```yaml
spec:
  ksmConfiguration:
    nodeLabelSelector:
      matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
            - node01
            - node03
```

- Enabling KSM on nodes with labels `kubevirt.io/first-label: true`, `kubevirt.io/second-label: true`:
```yaml
spec:
  ksmConfiguration:
    nodeLabelSelector:
      matchLabels:
        kubevirt.io/first-label: "true"
        kubevirt.io/second-label: "true"
```

- Enabling KSM on every node:
```yaml
spec:
  ksmConfiguration:
    nodeLabelSelector: {}
```

## Hyperconverged Kubevirt cluster-wide Crypto Policy API

Starting from OCP/OKD 4.6, a [cluster-wide API](https://github.com/openshift/enhancements/blob/master/enhancements/kube-apiserver/tls-config.md) is available for cluster administrators to set TLS profiles for OCP/OKD core components.
HCO, as an OCP/OKD layered product, will follow along OCP/OKD crypto policy cluster-wide setting, and use the same profile configured for the cluster’s control plane.
Configuration of a TLS security profile ensures that OCP/OKD, as well as HCO and its sibling operators and operands, use cryptographic libraries that do not allow known insecure protocols, ciphers, or algorithms.

By default, on OCP/OKD, HCO will read the global configuration for TLS security profile of the APIServer, without storing it in HCO CR, and will propagate the .spec.tlsSecurityProfile stanza to all underlying HCO managed custom resources.
The value on the HCO CR can be used to override the cluster wide setting.

The TLS security profiles are based on [Mozilla Recommended Configurations](https://wiki.mozilla.org/Security/Server_Side_TLS):
* `Old` - intended for use with legacy clients of libraries; requires a minimum TLS version of 1.0
* `Intermediate` - the default profile for all components; requires a minimum TLS version of 1.2
* `Modern` - intended for use with clients that don’t need backward compatibility; requires a minimum TLS version of 1.3. Unsupported in OCP/OKD 4.8 and below.

`Custom` profile allows you to define the TLS version and ciphers to use. Use caution when using a `Custom` profile, because invalid configurations can cause problems or make the control plane unreachable.
With the `Custom` profile, the cipher list should be expressed according to OpenSSL names.

On plain k8s, where APIServer CR is not available, the default value will be `Intermediate`.

## Configure Application Aware Quota (AAQ)
To enable the AAQ feature, set the `spec.enableApplicationAwareQuota` field to `true`.

To configure AAQ, set the fields of the `applicationAwareConfig` object in the HyperConverged resource's spec. The 
`applicationAwareConfig` object contains several fields:
* `vmiCalcConfigName` - determines how resource allocation will be done with ApplicationResourceQuota.
  Supported values are:
  * `VmiPodUsage` - calculates pod usage for VM-associated pods while concealing migration-specific resources. 
  * `VirtualResources` - allocates resources for VM-associated pods, using the VM's RAM size for memory and CPU threads 
     for processing.
  * `DedicatedVirtualResources` (default) - allocates resources for VM-associated pods, appending a /vm suffix to 
     `requests/limits.cpu` and `requests/limits.memory`, derived from the VM's RAM size and CPU threads. Notably, it 
     does not allocate resources for the standard `requests/limits.cpu` and `requests/limits.memory`.
  * `IgnoreVmiCalculator` - avoids allocating VM-associated pods differently from normal pods, maintaining uniform
     resource allocation.
    * `namespaceSelector` - determines in which namespaces AAQ's scheduling gate will be added to pods at creation time. 
     This field follows the standard Kubernetes selector format.
* `allowApplicationAwareClusterResourceQuota` (default = false) - set to true, to allow creation and management of ClusterAppsResourceQuota
  
  **note**: this setting cause some performance cost. Only set to true if there is a good reason.

### Limitations

Pods that set `.spec.nodeName` are not allowed to use in any namespace that AAQ is targeting.

### Example
```yaml
spec:
  enableApplicationAwareQuota: true
  applicationAwareConfig:
    vmiCalcConfigName: "VmiPodUsage"
    namespaceSelector:
      matchLabels:
        some-label: "some value"
    allowApplicationAwareClusterResourceQuota: true
```

**Note**: if a namespace selector is not being defined, AAQ would target namespaces with the `application-aware-quota/enable-gating` key.
It is also possible to target every namespace by defining an empty (non-nil) namespaceSelector as follows:
```yaml
spec:
  enableApplicationAwareQuota: true
  applicationAwareConfig:
    namespaceSelector: {}
```
## Configure higher workload density
Cluster administrators can opt-in for higher VM workload density by configuring the memroy overcommit percentage as follows:
```yaml
spec:
  higherWorkloadDensity:
    memoryOvercommitPercentage: <intValue> # i.e. 150, 200, default is 100
```
As an example, if the VM compute container requests 4Gi memory, then the corresponding VM guest OS will see `4Gi * 150 / 100 = 6Gi`.

**Note**: When updating the overcommit percentage, changes will apply to existing VM workloads only after a power cycle or after live-imgration. 

## Allow access to the Virtual Machine's VNC Console
In order to allow access toe the Virtual Machine's VNC Console, set the `spec.deployVmConsoleProxy` field to `true`.

SSP operator will deploy a proxy that provides the required access.

**Default**: `false`

### Example
```yaml
spec:
  deployVmConsoleProxy: true
```

## Configurations via Annotations

In addition to `featureGates` field in HyperConverged CR's spec, the user can set annotations in the HyperConverged CR
to unfold more configuration options.  
**Warning:** Annotations are less formal means of cluster configuration and may be dropped without the same deprecation
process of a regular API, such as in the `spec` section.

### OvS Opt-In Annotation

Starting from HCO version 1.3.0, OvS CNI support is disabled by default on new installations.  
In order to enable the deployment of OvS CNI DaemonSet on all _workload_ nodes, an annotation of `deployOVS: true` must
be set on HyperConverged CR.  
It can be set while creating the HyperConverged custom resource during the initial deployment, or during run time.

* To enable OvS CNI on the cluster, the HyperConverged CR should be similar to:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  annotations:
    deployOVS: "true"
...
```
* OvS CNI can also be enabled during run time of HCO, by annotating its CR:
```
kubectl annotate HyperConverged kubevirt-hyperconverged -n kubevirt-hyperconverged deployOVS=true --overwrite
```

If HCO was upgraded to 1.3.0 from a previous version, the annotation will be added as `true` and OvS will be deployed.  
Subsequent upgrades to newer versions will preserve the state from previous version, i.e. OvS will be deployed in the upgraded version if and only if it was deployed in the previous one.

### jsonpatch Annotations
HCO enables users to modify the operand CRs directly using jsonpatch annotations in HyperConverged CR.  
Modifications done to CRs using jsonpatch annotations won't be reconciled back by HCO to the opinionated defaults.  
The following annotations are supported in the HyperConverged CR:
* `kubevirt.kubevirt.io/jsonpatch` - for [KubeVirt configurations](https://github.com/kubevirt/api)
* `containerizeddataimporter.kubevirt.io/jsonpatch` - for [CDI configurations](https://github.com/kubevirt/containerized-data-importer-api)
* `networkaddonsconfigs.kubevirt.io/jsonpatch` - for [CNAO](https://github.com/kubevirt/cluster-network-addons-operator) configurations
* `ssp.kubevirt.io/jsonpatch` - for [SSP](https://github.com/kubevirt/ssp-operator) configurations

The content of the annotation will be a json array of patch objects, as defined in [RFC6902](https://tools.ietf.org/html/rfc6902).

#### Examples

##### Allow Post-Copy Migrations
The user wants to set the KubeVirt CR’s `spec.configuration.migrations.allowPostCopy` field to `true`. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    kubevirt.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/spec/configuration/migrations",
          "value": {"allowPostCopy": true}
        }
      ]
```

From CLI, it will be:
```bash
$ kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged \
  kubevirt.kubevirt.io/jsonpatch='[{"op": "add", \
    "path": "/spec/configuration/migrations", \
    "value": {"allowPostCopy": true} }]'
hyperconverged.hco.kubevirt.io/kubevirt-hyperconverged annotated
$ kubectl get kubevirt -n kubevirt-hyperconverged kubevirt-kubevirt-hyperconverged -o json \
  | jq '.spec.configuration.migrations.allowPostCopy'
true
$ kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged \
  kubevirt.kubevirt.io/jsonpatch='[{"op": "add", \
    "path": "/spec/configuration/migrations", \
    "value": {"allowPostCopy": false} }]'
hyperconverged.hco.kubevirt.io/kubevirt-hyperconverged annotated
$ kubectl get kubevirt -n kubevirt-hyperconverged kubevirt-kubevirt-hyperconverged -o json \
  | jq '.spec.configuration.migrations.allowPostCopy'
false
$ kubectl get hco -n kubevirt-hyperconverged  kubevirt-hyperconverged -o json \
  | jq '.status.conditions[] | select(.type == "TaintedConfiguration")'
{
  "lastHeartbeatTime": "2021-03-24T17:25:49Z",
  "lastTransitionTime": "2021-03-24T11:33:11Z",
  "message": "Unsupported feature was activated via an HCO annotation",
  "reason": "UnsupportedFeatureAnnotation",
  "status": "True",
  "type": "TaintedConfiguration"
}
```

##### Kubevirt Feature Gates
The user wants to enable experimental Kubevirt features
```yaml
metadata:
  annotations:
    kubevirt.kubevirt.io/jsonpatch: |-
      [
        {
          "op":"add",
          "path":"/spec/configuration/developerConfiguration/featureGates/-",
          "value":"CPUManager"
        }
      ]
```

##### Virt-handler Customization
The user wants to forcefully customize virt-handler configuration by setting custom values under `/spec/customizeComponents/patches` or the KV CR. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    kubevirt.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/spec/customizeComponents/patches",
          "value": [{
              "patch": "[{\"op\":\"add\",\"path\":\"/spec/template/spec/containers/0/command/-\",\"value\":\"--max-devices=250\"}]",
              "resourceName": "virt-handler",
              "resourceType": "Daemonset",
              "type": "json"
          }]
        }
      ]
```
##### Modify DataVolume Upload URL
The user wants to override the default URL used when uploading to a DataVolume, by setting the CDI CR's `spec.config.uploadProxyURLOverride` to `myproxy.example.com`. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    containerizeddataimporter.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/spec/config/uploadProxyURLOverride",
          "value": "myproxy.example.com"
        }
      ]
```

##### Alter the tlsSecurityProfile of a single component
You can potentially alter tlsSecurityProfile of a single component, please be aware that a bad configuration could potentially break the cluster.
Please notice that the old/intermediate/modern/custom stanza that you don't need should be explicitly set to `null` as part of the json-patch.
Please notice that Kubevirt uses a different structure.
```bash
kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged 'containerizeddataimporter.kubevirt.io/jsonpatch=[{"op": "replace", "path": "/spec/config/tlsSecurityProfile", "value": {"old":{}, "type": "Old", "intermediate": null, "modern": null, "custom": null  }}]'
kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged 'networkaddonsconfigs.kubevirt.io/jsonpatch=[{"op": "replace","path": "/spec/tlsSecurityProfile", "value": {"old":{}, "type": "Old", "intermediate": null, "modern": null, "custom": null  }}]'
kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged 'kubevirt.kubevirt.io/jsonpatch=[{"op": "replace","path": "/spec/configuration/tlsConfiguration", "value": {"ciphers": ["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256", "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256", "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA", "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA", "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA", "TLS_RSA_WITH_AES_128_GCM_SHA256", "TLS_RSA_WITH_AES_256_GCM_SHA384", "TLS_RSA_WITH_AES_128_CBC_SHA256", "TLS_RSA_WITH_AES_128_CBC_SHA", "TLS_RSA_WITH_AES_256_CBC_SHA", "TLS_RSA_WITH_3DES_EDE_CBC_SHA"], "minTLSVersion": "VersionTLS10" }}]'
```

##### Disable KubeMacPool
If KubeMacPool is buggy on your cluster and you do not immediately need it you can ask CNAO not to deploy it
```bash
kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged 'networkaddonsconfigs.kubevirt.io/jsonpatch=[{"op": "replace","path": "/spec/kubeMacPool","value": null}]'
```

**_Note:_** The full configurations options for Kubevirt, CDI and CNAO which are available on the cluster, can be explored by using `kubectl explain <resource name>.spec`. For example:  
```bash
$ kubectl explain kv.spec
KIND:     KubeVirt
VERSION:  kubevirt.io/v1

RESOURCE: spec <Object>

DESCRIPTION:
     <empty>

FIELDS:
   certificateRotateStrategy	<Object>

   configuration	<Object>
     holds kubevirt configurations. same as the virt-configMap

   customizeComponents	<Object>

   imagePullPolicy	<string>
     The ImagePullPolicy to use.

   imageRegistry	<string>
     The image registry to pull the container images from Defaults to the same
     registry the operator's container image is pulled from.
  
  <truncated>
```

To inspect lower-level objects under `spec`, they can be specified in `kubectl explain`, recursively. e.g.  
```bash
$ kubectl explain kv.spec.configuration.network
KIND:     KubeVirt
VERSION:  kubevirt.io/v1

RESOURCE: network <Object>

DESCRIPTION:
     NetworkConfiguration holds network options

FIELDS:
   defaultNetworkInterface	<string>

   permitBridgeInterfaceOnPodNetwork	<boolean>

   permitSlirpInterface	<boolean>
```

* To explore kubevirt configuration options, use `kubectl explain kv.spec`
* To explore CDI configuration options, use `kubectl explain cdi.spec`
* To explore CNAO configuration options, use `kubectl explain networkaddonsconfig.spec`
* To explore SSP configuration options, use `kubectl explain ssp.spec`

### WARNING
Using the jsonpatch annotation feature incorrectly might lead to unexpected results and could potentially render the Kubevirt-Hyperconverged system unstable.  
The jsonpatch annotation feature is particularly dangerous when upgrading Kubevirt-Hyperconverged, as the structure or the semantics of the underlying components' CR might be changed. Please remove any jsonpatch annotation usage prior the upgrade, to avoid any potential issues.
**USE WITH CAUTION!**

As the usage of the jsonpatch annotation is not safe, the HyperConverged Cluster Operator will count the number of these
modifications in a metric named kubevirt_hco_unsafe_modifications.
if the counter is not zero, an alert named
`UnsupportedHCOModification will` be eventually fired:
```
Labels
    alertname=UnsupportedHCOModification
    annotation_name="kubevirt.kubevirt.io/jsonpatch"
    severity=info
```

## Tune Kubevirt Rate Limits
Kubevirt API clients come with a token bucket rate limiter which avoids to congest the kube-apiserver bandwidth.
The rate limiters are configurable through `burst` and `Query Per Second (QPS)` parameters.
Whilst the rate limiter may avoid congestion, it may also limit the number of VMs that can be deployed in the cluster.
Therefore, HCO enables the feature `tuningPolicy` for allowing to tune the rate limiters parameters.
Currently, there are two profiles supported: `annotation` and `highBurst`.

### Annotation Profile

The `tuningPolicy` profile `annotation` is intended for arbitrary `burst` and `QPS` values, i.e. the values are fully
configurable with the desired ones.
By using this profile, the user is responsible for setting the values more appropriated to its particular scenario.

> **_Note_:** If no `tuningPolicy` is configured or the `tuningPolicy` feature is not well configured, Kubevirt will use the
> [default](https://github.com/kubevirt/kubevirt/blob/a3e92eb499636cbab46763fbdd1dbccaca716c29/pkg/virt-config/virt-config.go#L78-L86) rate limiter values.

The `annotation` policy relies on the annotation `hco.kubevirt.io/tuningPolicy` to specify the desired values of `burst` and `QPS` parameters.
The structure of the annotation is the following one:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
  annotations:
    hco.kubevirt.io/tuningPolicy: '{"qps":100,"burst":200}'
...
spec:
  tuningPolicy: annotation
```

Where the values of `qps` and `burst` can be replaced by any desired value of `QPS` and `burst`, respectively.
For instance, in the above example the `QPS` parameter is set to 100 and the `burst` parameter is set to 200.
The annotation can be created with the following command:

```bash
kubectl annotate -n kubevirt-hyperconverged hco kubevirt-hyperconverged hco.kubevirt.io/tuningPolicy='{"qps":100,"burst":200}'
```

> **_Note_:**  HCO will not configure the rate limiters if both the annotation and the `spec.tuningPolicy` are not populated correctly.
> Moreover, if `spec.tuningPolicy` is set but the annotation is not present, HCO will reject the changes in the HyperConverged CR.
> In case that the annotation is defined but the `spec.tuningPolicy` is not set, HCO will ignore the rate limit configurations.

The `tuningPolicy` feature can be enabled using the following patch:

```bash
kubectl patch -n kubevirt-hyperconverged hco kubevirt-hyperconverged --type=json -p='[{"op": "add", "path": "/spec/tuningPolicy", "value": "annotation"}]'
```

### HighBurst Profile

The `highBurst` profile is intended for high load scenarios where the user expect to create and maintain a high number of VMs
in the same cluster.
The profile configures internally the more suitable `burst` and `QPS` values for the most common high load scenarios. 
Nevertheless, the specific configuration of those values is hidden to the user.
Also, the values may change over time since they are based on an experimentation process.

To enable this `tuningPolicy` profile, the following patch may be applied:

```bash
kubectl patch -n kubevirt-hyperconverged hco kubevirt-hyperconverged --type=json -p='[{"op": "add", "path": "/spec/tuningPolicy", "value": "highBurst"}]'
```

## Kube Descheduler integration
A [Descheduler](https://github.com/kubernetes-sigs/descheduler) is a Kubernetes application that causes the control plane to re-arrange the workloads in a better way.
It operates every pre-defined period and goes back to sleep after it had performed its job.

The descheduler uses the Kubernetes [eviction API](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/) to evict pods, and receives feedback from `kube-api` whether the eviction request was granted or not.
On the other side, in order to keep VM live and trigger live-migration, KubeVirt handles eviction requests in a custom way and unfortunately a live migration takes time.
So from the descheduler's point of view, `virt-launcher` pods fail to be evicted, but they actually migrating to another node in background.
The descheduler notes the failure to evict the `virt-launcher` pod and keeps trying to evict other pods, typically resulting in it attempting to evict substantially all `virt-launcher` pods from the node triggering a migration storm.
In other words, the way KubeVirt handles eviction requests causes the descheduler to make wrong decisions and take wrong actions that could destabilize the cluster.
Using the descheduler operator with the `LowNodeUtilization` strategy results in unstable/oscillatory behavior if the descheduler is used in this way to migrate VMs.
To correctly handle the special case of `VM` pod evicted triggering a live migration to another node, the `Kube Descheduler Operator` introduced a `profileCustomizations` named `devEnableEvictionsInBackground`
which is currently considered an `alpha` [feature](https://github.com/kubernetes-sigs/descheduler/tree/master/keps/1397-evictions-in-background) on `Kube Descheduler Operator` side.
to prevent unexpected behaviours, if the `Kube Descheduler Operator` is installed and configured alongside `HCO`, `HCO` will check its configuration looking for the presence of `devEnableEvictionsInBackground` `profileCustomizations` eventually
suggesting to the cluster admin to fix the configuration of the `Kube Descheduler Operator` via an `alert` and its linked `runbook`.

In order to fix the configuration of the `Kube Descheduler Operator` to be suitable also for the KubeVirt use case,
something like:
```yaml
apiVersion: operator.openshift.io/v1
kind: KubeDescheduler
metadata:
  name: cluster
  namespace: openshift-kube-descheduler-operator
spec:
  profileCustomizations:
    devEnableEvictionsInBackground: true
```
should be merged in its configuration.

## KubeVirt Instance Type Configuration

The configuration of [instance type and preference related features](https://kubevirt.io/user-guide/user_workloads/instancetypes/) within KubeVirt can be configured through the use of the `spec.instancetypeConfig` configurable. This field matches the field provided by the [KubeVirt CR](https://kubevirt.io/api-reference/main/definitions.html#_v1_instancetypeconfiguration).

The following example configures an [InstancetypeReferencePolicy](https://kubevirt.io/user-guide/user_workloads/instancetypes/#instancetypereferencepolicy) of `expand`:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  instancetypeConfig:
    referencePolicy: expand
```
## KubeVirt Common Instance Types Deployment Configuration

The configuration of [common instance types deployment](https://kubevirt.io/user-guide/user_workloads/deploy_common_instancetypes/) is held separately to the core instance type and preference functionality within the [KubeVirt CR](https://kubevirt.io/api-reference/main/definitions.html#_v1_commoninstancetypesdeployment). This is mirrored in the HCO CR through the `spec.commonInstancetypesDeployment` configurable.

The following example disables the deployment of common instance types within the cluster:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  commonInstancetypesDeployment:
    enabled: false

```

## KubeVirt Live Update Configuration

In order to configure the live upgrade of virtual machines, you can set the optional `spec.liveUpdateConfiguration` object in the HyperConverged CR. See more details in the [KubeVirt documentation](https://kubevirt.io/user-guide/compute/cpu_hotplug/#optional-set-maximum-sockets-or-hotplug-ratio).
The `spec.liveUpdateConfiguration` field is with the same structure as the corresponding field in the [KubeVirt CR](https://kubevirt.io/api-reference/main/definitions.html#_v1_liveupdateconfiguration). 

The following example sets cluster configuration for hotplug ratio, max cpu socket and max guest size.

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  liveUpdateConfiguration:
    maxHotplugRatio: 3
    maxCpuSockets: 2
    maxGuest: 2Gi
```
