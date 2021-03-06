package acsengine

import (
	"fmt"
	"net"

	"github.com/Azure/acs-engine/pkg/api"
)

var (
	//AzureCloudSpec is the default configurations for global azure.
	AzureCloudSpec = AzureEnvironmentSpecConfig{
		//DockerSpecConfig specify the docker engine download repo
		DockerSpecConfig: DockerSpecConfig{
			DockerEngineRepo: "https://aptdocker.azureedge.net/repo",
		},
		//KubernetesSpecConfig is the default kubernetes container image url.
		KubernetesSpecConfig: KubernetesSpecConfig{
			KubernetesImageBase:    "gcrio.azureedge.net/google_containers/",
			TillerImageBase:        "gcrio.azureedge.net/kubernetes-helm/",
			KubeBinariesSASURLBase: "https://acs-mirror.azureedge.net/wink8s/",
		},

		DCOSSpecConfig: DCOSSpecConfig{
			DCOS173BootstrapDownloadURL: fmt.Sprintf(MsecndDCOSBootstrapDownloadURL, "testing", "df308b6fc3bd91e1277baa5a3db928ae70964722"),
			DCOS184BootstrapDownloadURL: fmt.Sprintf(AzureEdgeDCOSBootstrapDownloadURL, "testing", "5b4aa43610c57ee1d60b4aa0751a1fb75824c083"),
			DCOS187BootstrapDownloadURL: fmt.Sprintf(AzureEdgeDCOSBootstrapDownloadURL, "stable", "e73ba2b1cd17795e4dcb3d6647d11a29b9c35084"),
			DCOS188BootstrapDownloadURL: fmt.Sprintf(AzureEdgeDCOSBootstrapDownloadURL, "stable", "5df43052907c021eeb5de145419a3da1898c58a5"),
			DCOS190BootstrapDownloadURL: fmt.Sprintf(AzureEdgeDCOSBootstrapDownloadURL, "stable", "58fd0833ce81b6244fc73bf65b5deb43217b0bd7"),
		},
	}

	//AzureChinaCloudSpec is the configurations for Azure China (Mooncake)
	AzureChinaCloudSpec = AzureEnvironmentSpecConfig{
		//DockerSpecConfig specify the docker engine download repo
		DockerSpecConfig: DockerSpecConfig{
			DockerEngineRepo: "https://mirror.azure.cn/docker-engine/apt/repo/",
		},
		//KubernetesSpecConfig - Due to Chinese firewall issue, the default containers from google is blocked, use the Chinese local mirror instead
		KubernetesSpecConfig: KubernetesSpecConfig{
			KubernetesImageBase:    "mirror.azure.cn:5000/google_containers/",
			KubeBinariesSASURLBase: "https://acs-mirror.azureedge.net/wink8s/",
		},
		DCOSSpecConfig: DCOSSpecConfig{
			DCOS173BootstrapDownloadURL: fmt.Sprintf(AzureChinaCloudDCOSBootstrapDownloadURL, "df308b6fc3bd91e1277baa5a3db928ae70964722"),
			DCOS184BootstrapDownloadURL: fmt.Sprintf(AzureChinaCloudDCOSBootstrapDownloadURL, "5b4aa43610c57ee1d60b4aa0751a1fb75824c083"),
			DCOS187BootstrapDownloadURL: fmt.Sprintf(AzureChinaCloudDCOSBootstrapDownloadURL, "e73ba2b1cd17795e4dcb3d6647d11a29b9c35084"),
			DCOS188BootstrapDownloadURL: fmt.Sprintf(AzureChinaCloudDCOSBootstrapDownloadURL, "5df43052907c021eeb5de145419a3da1898c58a5"),
		},
	}
)

// SetPropertiesDefaults for the container Properties, returns true if certs are generated
func SetPropertiesDefaults(cs *api.ContainerService) (bool, error) {
	properties := cs.Properties

	setOrchestratorDefaults(cs)

	setMasterNetworkDefaults(properties)

	setAgentNetworkDefaults(properties)

	setStorageDefaults(properties)

	certsGenerated, e := setDefaultCerts(properties)
	if e != nil {
		return false, e
	}
	return certsGenerated, nil
}

// setOrchestratorDefaults for orchestrators
func setOrchestratorDefaults(cs *api.ContainerService) {
	location := cs.Location
	a := cs.Properties

	cloudSpecConfig := GetCloudSpecConfig(location)
	if a.OrchestratorProfile.OrchestratorType == api.Kubernetes {
		k8sVersion := a.OrchestratorProfile.OrchestratorVersion
		if a.OrchestratorProfile.KubernetesConfig == nil {
			a.OrchestratorProfile.KubernetesConfig = &api.KubernetesConfig{}
		}
		a.OrchestratorProfile.KubernetesConfig.KubernetesImageBase = cloudSpecConfig.KubernetesSpecConfig.KubernetesImageBase
		if a.OrchestratorProfile.KubernetesConfig.NetworkPolicy == "" {
			a.OrchestratorProfile.KubernetesConfig.NetworkPolicy = DefaultNetworkPolicy
		}
		if a.OrchestratorProfile.KubernetesConfig.ClusterSubnet == "" {
			if a.OrchestratorProfile.IsVNETIntegrated() {
				// When VNET integration is enabled, all masters, agents and pods share the same large subnet.
				a.OrchestratorProfile.KubernetesConfig.ClusterSubnet = DefaultKubernetesSubnet
			} else {
				a.OrchestratorProfile.KubernetesConfig.ClusterSubnet = DefaultKubernetesClusterSubnet
			}
		}
		if a.OrchestratorProfile.KubernetesConfig.DockerBridgeSubnet == "" {
			a.OrchestratorProfile.KubernetesConfig.DockerBridgeSubnet = DefaultDockerBridgeSubnet
		}
		if a.OrchestratorProfile.KubernetesConfig.NodeStatusUpdateFrequency == "" {
			a.OrchestratorProfile.KubernetesConfig.NodeStatusUpdateFrequency = KubeImages[k8sVersion]["nodestatusfreq"]
		}
		if a.OrchestratorProfile.KubernetesConfig.CtrlMgrNodeMonitorGracePeriod == "" {
			a.OrchestratorProfile.KubernetesConfig.CtrlMgrNodeMonitorGracePeriod = KubeImages[k8sVersion]["nodegraceperiod"]
		}
		if a.OrchestratorProfile.KubernetesConfig.CtrlMgrPodEvictionTimeout == "" {
			a.OrchestratorProfile.KubernetesConfig.CtrlMgrPodEvictionTimeout = KubeImages[k8sVersion]["podeviction"]
		}
		if a.OrchestratorProfile.KubernetesConfig.CtrlMgrRouteReconciliationPeriod == "" {
			a.OrchestratorProfile.KubernetesConfig.CtrlMgrRouteReconciliationPeriod = KubeImages[k8sVersion]["routeperiod"]
		}
		// Enforce sane cloudprovider backoff defaults, if CloudProviderBackoff is true in KubernetesConfig
		if a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoff == true {
			if a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffDuration == 0 {
				a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffDuration = DefaultKubernetesCloudProviderBackoffDuration
			}
			if a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffExponent == 0 {
				a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffExponent = DefaultKubernetesCloudProviderBackoffExponent
			}
			if a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffJitter == 0 {
				a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffJitter = DefaultKubernetesCloudProviderBackoffJitter
			}
			if a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffRetries == 0 {
				a.OrchestratorProfile.KubernetesConfig.CloudProviderBackoffRetries = DefaultKubernetesCloudProviderBackoffRetries
			}
		}
		// Enforce sane cloudprovider rate limit defaults, if CloudProviderRateLimit is true in KubernetesConfig
		if a.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimit == true && (k8sVersion == api.Kubernetes172 || k8sVersion == api.Kubernetes171 || k8sVersion == api.Kubernetes170 || k8sVersion == api.Kubernetes166) {
			if a.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitQPS == 0 {
				a.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitQPS = DefaultKubernetesCloudProviderRateLimitQPS
			}
			if a.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket == 0 {
				a.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket = DefaultKubernetesCloudProviderRateLimitBucket
			}
		}
	}
}

// SetMasterNetworkDefaults for masters
func setMasterNetworkDefaults(a *api.Properties) {
	if !a.MasterProfile.IsCustomVNET() {
		if a.OrchestratorProfile.OrchestratorType == api.Kubernetes {
			if a.OrchestratorProfile.IsVNETIntegrated() {
				// When VNET integration is enabled, all masters, agents and pods share the same large subnet.
				a.MasterProfile.Subnet = a.OrchestratorProfile.KubernetesConfig.ClusterSubnet
				a.MasterProfile.FirstConsecutiveStaticIP = getFirstConsecutiveStaticIPAddress(a.MasterProfile.Subnet)
			} else {
				a.MasterProfile.Subnet = DefaultKubernetesMasterSubnet
				a.MasterProfile.FirstConsecutiveStaticIP = DefaultFirstConsecutiveKubernetesStaticIP
			}
		} else if a.HasWindows() {
			a.MasterProfile.Subnet = DefaultSwarmWindowsMasterSubnet
			a.MasterProfile.FirstConsecutiveStaticIP = DefaultSwarmWindowsFirstConsecutiveStaticIP
		} else {
			a.MasterProfile.Subnet = DefaultMasterSubnet
			a.MasterProfile.FirstConsecutiveStaticIP = DefaultFirstConsecutiveStaticIP
		}
	}

	// Allocate IP addresses for containers if VNET integration is enabled.
	// A custom count specified by the user overrides this value.
	if a.MasterProfile.IPAddressCount == 0 {
		if a.OrchestratorProfile.IsVNETIntegrated() {
			a.MasterProfile.IPAddressCount = DefaultAgentMultiIPAddressCount
		} else {
			a.MasterProfile.IPAddressCount = DefaultAgentIPAddressCount
		}
	}

	if a.MasterProfile.HttpSourceAddressPrefix == "" {
		a.MasterProfile.HttpSourceAddressPrefix = "*"
	}
}

// SetAgentNetworkDefaults for agents
func setAgentNetworkDefaults(a *api.Properties) {
	// configure the subnets if not in custom VNET
	if !a.MasterProfile.IsCustomVNET() {
		subnetCounter := 0
		for _, profile := range a.AgentPoolProfiles {
			if a.OrchestratorProfile.OrchestratorType == api.Kubernetes {
				profile.Subnet = a.MasterProfile.Subnet
			} else {
				profile.Subnet = fmt.Sprintf(DefaultAgentSubnetTemplate, subnetCounter)
			}

			subnetCounter++
		}
	}

	for _, profile := range a.AgentPoolProfiles {
		// set default OSType to Linux
		if profile.OSType == "" {
			profile.OSType = api.Linux
		}

		// Allocate IP addresses for containers if VNET integration is enabled.
		// A custom count specified by the user overrides this value.
		if profile.IPAddressCount == 0 {
			if a.OrchestratorProfile.IsVNETIntegrated() {
				profile.IPAddressCount = DefaultAgentMultiIPAddressCount
			} else {
				profile.IPAddressCount = DefaultAgentIPAddressCount
			}
		}
	}
}

// setStorageDefaults for agents
func setStorageDefaults(a *api.Properties) {
	if len(a.MasterProfile.StorageProfile) == 0 {
		a.MasterProfile.StorageProfile = api.StorageAccount
	}
	for _, profile := range a.AgentPoolProfiles {
		if len(profile.StorageProfile) == 0 {
			profile.StorageProfile = api.StorageAccount
		}
		if len(profile.AvailabilityProfile) == 0 {
			profile.AvailabilityProfile = api.VirtualMachineScaleSets
		}
	}
}

func setDefaultCerts(a *api.Properties) (bool, error) {
	if !certGenerationRequired(a) {
		return false, nil
	}

	masterExtraFQDNs := FormatAzureProdFQDNs(a.MasterProfile.DNSPrefix)
	firstMasterIP := net.ParseIP(a.MasterProfile.FirstConsecutiveStaticIP).To4()

	if firstMasterIP == nil {
		return false, fmt.Errorf("MasterProfile.FirstConsecutiveStaticIP '%s' is an invalid IP address", a.MasterProfile.FirstConsecutiveStaticIP)
	}

	ips := []net.IP{firstMasterIP}

	// Add the Internal Loadbalancer IP which is always at at a known offset from the firstMasterIP
	ips = append(ips, net.IP{firstMasterIP[0], firstMasterIP[1], firstMasterIP[2], firstMasterIP[3] + byte(DefaultInternalLbStaticIPOffset)})

	// Include the Internal load balancer as well
	for i := 1; i < a.MasterProfile.Count; i++ {
		ip := net.IP{firstMasterIP[0], firstMasterIP[1], firstMasterIP[2], firstMasterIP[3] + byte(i)}
		ips = append(ips, ip)
	}

	if a.CertificateProfile == nil {
		a.CertificateProfile = &api.CertificateProfile{}
	}

	// use the specified Certificate Authority pair, or generate a new pair
	var caPair *PkiKeyCertPair
	if len(a.CertificateProfile.CaCertificate) != 0 && len(a.CertificateProfile.CaPrivateKey) != 0 {
		caPair = &PkiKeyCertPair{CertificatePem: a.CertificateProfile.CaCertificate, PrivateKeyPem: a.CertificateProfile.CaPrivateKey}
	} else {
		caCertificate, caPrivateKey, err := createCertificate("ca", nil, nil, false, nil, nil)
		if err != nil {
			return false, err
		}
		caPair = &PkiKeyCertPair{CertificatePem: string(certificateToPem(caCertificate.Raw)), PrivateKeyPem: string(privateKeyToPem(caPrivateKey))}
		a.CertificateProfile.CaCertificate = caPair.CertificatePem
		a.CertificateProfile.CaPrivateKey = caPair.PrivateKeyPem
	}

	apiServerPair, clientPair, kubeConfigPair, err := CreatePki(masterExtraFQDNs, ips, DefaultKubernetesClusterDomain, caPair)
	if err != nil {
		return false, err
	}

	a.CertificateProfile.APIServerCertificate = apiServerPair.CertificatePem
	a.CertificateProfile.APIServerPrivateKey = apiServerPair.PrivateKeyPem
	a.CertificateProfile.ClientCertificate = clientPair.CertificatePem
	a.CertificateProfile.ClientPrivateKey = clientPair.PrivateKeyPem
	a.CertificateProfile.KubeConfigCertificate = kubeConfigPair.CertificatePem
	a.CertificateProfile.KubeConfigPrivateKey = kubeConfigPair.PrivateKeyPem

	return true, nil
}

func certGenerationRequired(a *api.Properties) bool {
	if a.CertificateProfile != nil &&
		(len(a.CertificateProfile.APIServerCertificate) > 0 || len(a.CertificateProfile.APIServerPrivateKey) > 0 ||
			len(a.CertificateProfile.ClientCertificate) > 0 || len(a.CertificateProfile.ClientPrivateKey) > 0) {
		return false
	}

	switch a.OrchestratorProfile.OrchestratorType {
	case api.DCOS:
		return false
	case api.Swarm:
		return false
	case api.SwarmMode:
		return false
	case api.Kubernetes:
		return true
	default:
		return false
	}
}

// getFirstConsecutiveStaticIPAddress returns the first static IP address of the given subnet.
func getFirstConsecutiveStaticIPAddress(subnetStr string) string {
	_, subnet, err := net.ParseCIDR(subnetStr)
	if err != nil {
		return DefaultFirstConsecutiveKubernetesStaticIP
	}

	// Find the first and last octet of the host bits.
	ones, bits := subnet.Mask.Size()
	firstOctet := ones / 8
	lastOctet := bits/8 - 1

	// Set the remaining host bits in the first octet.
	subnet.IP[firstOctet] |= (1 << byte((8 - (ones % 8)))) - 1

	// Fill the intermediate octets with 1s and last octet with offset. This is done so to match
	// the existing behavior of allocating static IP addresses from the last /24 of the subnet.
	for i := firstOctet + 1; i < lastOctet; i++ {
		subnet.IP[i] = 255
	}
	subnet.IP[lastOctet] = DefaultKubernetesFirstConsecutiveStaticIPOffset

	return subnet.IP.String()
}
