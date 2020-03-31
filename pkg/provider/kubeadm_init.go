// Copyright © 2019 Alvaro Saurin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"fmt"
	"net"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// dataSourceToInitConfig copies some settings from the
// Terraform `data` definition to a kubeadm Init configuration
func dataSourceToInitConfig(d *schema.ResourceData, token string) (*kubeadmapi.InitConfiguration, error) {
	ssh.Debug("creating initialization configuration...")

	initConfig := &kubeadmapi.InitConfiguration{
		ClusterConfiguration: kubeadmapi.ClusterConfiguration{
			// FeatureGates:         featureGates,
			APIServer: kubeadmapi.APIServer{
				CertSANs: []string{},
			},
		},
		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: common.DefKubeletSettings,
		},
	}

	if _, ok := d.GetOk("api.0"); ok {
		if external, ok := d.GetOk("api.0.external"); ok {
			initConfig.ControlPlaneEndpoint = common.AddressWithPort(external.(string), common.DefAPIServerPort)
		}

		if internal, ok := d.GetOk("api.0.internal"); ok {
			host, port, err := net.SplitHostPort(internal.(string))
			if err != nil {
				return nil, err
			}

			initConfig.LocalAPIEndpoint.AdvertiseAddress = host
			if port != "" {
				i, err := strconv.Atoi(port)
				if err != nil {
					return nil, err
				}
				initConfig.LocalAPIEndpoint.BindPort = int32(i)
			}

			initConfig.ClusterConfiguration.APIServer.CertSANs = append(initConfig.ClusterConfiguration.APIServer.CertSANs, host)
		}

		if altNames, ok := d.GetOk("api.0.alt_names"); ok {
			initConfig.APIServer.CertSANs = append(initConfig.APIServer.CertSANs, altNames.([]string)...)
		}
	}

	if _, ok := d.GetOk("network.0"); ok {
		if podCIDROpt, ok := d.GetOk("network.0.pods"); ok {
			initConfig.Networking.PodSubnet = podCIDROpt.(string)
		}
		if servicesCIDROpt, ok := d.GetOk("network.0.services"); ok {
			initConfig.Networking.ServiceSubnet = servicesCIDROpt.(string)
		}

		if _, ok := d.GetOk("network.0.dns.0"); ok {
			if dnsDomainOpt, ok := d.GetOk("network.0.dns.0.domain"); ok {
				dnsDomain := dnsDomainOpt.(string)

				// validate the DNS domain... otherwise we will get an error when
				// we run `kubeadm init`
				r, _ := regexp.Compile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`)
				if !r.MatchString(dnsDomain) {
					return nil, fmt.Errorf("invalid DNS name '%s': a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com')", dnsDomain)
				}

				initConfig.Networking.DNSDomain = dnsDomain
			}

			if dnsUpstreamOpt, ok := d.GetOk("network.0.dns.0.upstream"); ok {
				dnsUp := dnsUpstreamOpt.([]interface{})
				if len(dnsUp) > 0 {
					initConfig.NodeRegistration.KubeletExtraArgs["resolv-conf"] = common.DefResolvUpstreamConf
				}
			}
		}
	}

	if _, ok := d.GetOk("images.0"); ok {
		kube_repo := d.Get("images.0.kube_repo").(string)
		initConfig.ClusterConfiguration.ImageRepository = kube_repo

		etcd_repo := d.Get("images.0.etcd_repo").(string)
		etcd_version := d.Get("images.0.etcd_version").(string)
		if etcd_version != "" || etcd_repo != "" {
			initConfig.ClusterConfiguration.Etcd = kubeadmapi.Etcd{
				Local: &kubeadmapi.LocalEtcd{
					ImageMeta: kubeadmapi.ImageMeta{
						ImageRepository: etcd_repo,
						ImageTag:        etcd_version,
					},
				},
			}
		}
	}

	if _, ok := d.GetOk("runtime.0"); ok {
		if runtimeEngineOpt, ok := d.GetOk("runtime.0.engine"); ok {
			if socket, ok := common.DefCriSocket[runtimeEngineOpt.(string)]; ok {
				ssh.Debug("setting CRI socket '%s'", socket)
				initConfig.NodeRegistration.KubeletExtraArgs["container-runtime-endpoint"] = fmt.Sprintf("unix://%s", socket)
				initConfig.NodeRegistration.CRISocket = socket
			} else {
				return nil, fmt.Errorf("unknown runtime engine %s", runtimeEngineOpt.(string))
			}
		}

		if _, ok := d.GetOk("runtime.0.extra_args.0"); ok {
			if args, ok := d.GetOk("runtime.0.extra_args.0.api_server"); ok {
				initConfig.ClusterConfiguration.APIServer.ExtraArgs = args.(map[string]string)
			}
			if args, ok := d.GetOk("runtime.0.extra_args.0.controller_manager"); ok {
				initConfig.ClusterConfiguration.ControllerManager.ExtraArgs = args.(map[string]string)
			}
			if args, ok := d.GetOk("runtime.0.extra_args.0.scheduler"); ok {
				initConfig.ClusterConfiguration.Scheduler.ExtraArgs = args.(map[string]string)
			}
			if args, ok := d.GetOk("runtime.0.extra_args.0.kubelet"); ok {
				initConfig.NodeRegistration.KubeletExtraArgs = args.(map[string]string)
			}
		}
	}

	// check if we have some cloud-provider
	// if that is the case, we use the "external" cloud provider.
	// the provisioner will have to load a "manifest" for running this externla cloud provider manager
	if cloudProvRaw, ok := d.GetOk("cloud.0.provider"); ok && len(cloudProvRaw.(string)) > 0 {
		if initConfig.NodeRegistration.KubeletExtraArgs == nil {
			initConfig.NodeRegistration.KubeletExtraArgs = map[string]string{}
		}
		initConfig.NodeRegistration.KubeletExtraArgs["cloud-provider"] = "external"

		if initConfig.ClusterConfiguration.APIServer.ExtraArgs == nil {
			initConfig.ClusterConfiguration.APIServer.ExtraArgs = map[string]string{}
		}
		initConfig.ClusterConfiguration.APIServer.ExtraArgs["cloud-provider"] = "external"

		if initConfig.ClusterConfiguration.ControllerManager.ExtraArgs == nil {
			initConfig.ClusterConfiguration.ControllerManager.ExtraArgs = map[string]string{}
		}
		initConfig.ClusterConfiguration.ControllerManager.ExtraArgs["cloud-provider"] = "external"
	}

	if _, ok := d.GetOk("cni.0"); ok {
		if arg, ok := d.GetOk("cni.0.bin_dir"); ok {
			initConfig.NodeRegistration.KubeletExtraArgs["cni-bin-dir"] = arg.(string)
		}
		if arg, ok := d.GetOk("cni.0.conf_dir"); ok {
			initConfig.NodeRegistration.KubeletExtraArgs["cni-conf-dir"] = arg.(string)
		}
	}

	if versionOpt, ok := d.GetOk("version"); ok && len(versionOpt.(string)) > 0 {
		initConfig.KubernetesVersion = versionOpt.(string)
	}

	if _, ok := d.GetOk("etcd.0"); ok {
		if etcdServersLst, ok := d.GetOk("etcd.0.endpoints"); ok {
			if initConfig.Etcd.External == nil {
				initConfig.Etcd.External = &kubeadmapi.ExternalEtcd{}
			}
			initConfig.Etcd.External.Endpoints = etcdServersLst.([]string)
		}
	}

	if len(token) > 0 {
		t, err := common.NewBootstrapToken(token)
		if err != nil {
			return nil, err
		}
		t.Expires = nil
		initConfig.BootstrapTokens = []kubeadmapi.BootstrapToken{t}
	}

	return initConfig, nil
}
