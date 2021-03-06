// Copyright Jetstack Ltd. See LICENSE for details.
package interfaces

import (
	"context"
	"io"
	"net"
	"os"

	vault "github.com/hashicorp/vault/api"
	"github.com/jetstack/vault-unsealer/pkg/kv"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	clusterv1alpha1 "github.com/jetstack/tarmak/pkg/apis/cluster/v1alpha1"
	tarmakv1alpha1 "github.com/jetstack/tarmak/pkg/apis/tarmak/v1alpha1"
	"github.com/jetstack/tarmak/pkg/tarmak/role"
	"github.com/jetstack/tarmak/pkg/tarmak/utils/input"
	wingclient "github.com/jetstack/tarmak/pkg/wing/client/clientset/versioned"
)

type Cluster interface {
	Variables() map[string]interface{}
	Environment() Environment
	Name() string
	NetworkCIDR() *net.IPNet
	RemoteState() string

	// get the absolute config path to cluster's config folder
	ConfigPath() string

	Config() *clusterv1alpha1.Cluster
	Images() []string // This returns all neccessary base images
	SSHConfigPath() string
	SSHHostKeysPath() string
	ClusterName() string
	Log() *logrus.Entry
	APITunnel() Tunnel
	Region() string
	Subnets() []clusterv1alpha1.Subnet // Return subnets per AZ
	Role(string) *role.Role
	Roles() []*role.Role
	InstancePools() []InstancePool
	InstancePool(string) InstancePool
	ImageIDs() (map[string]string, error)
	Parameters() map[string]string
	Type() string
	ListHosts() ([]Host, error)
	// This enforces a reapply of the puppet.tar.gz on every instance in the cluster
	ReapplyConfiguration() error
	// This waits until all instances have congverged successfully
	WaitForConvergance() error
	// This upload the puppet.tar.gz to the cluster, warning there is some duplication as terraform is also uploading this puppet.tar.gz
	UploadConfiguration() error
	// Verify the cluster (these contain more expensive calls like AWS calls
	Verify() error
	// Validate the cluster (these contain less expensive local calls)
	Validate() error

	// This state is either destroy or apply
	GetState() string
	SetState(string)

	// get the terrform output for this cluster
	TerraformOutput() (map[string]interface{}, error)

	// return api hostnames
	PublicAPIHostname() string
        PrivateAPIHostname() string

	// cluster uses encrypted EBS
	AmazonEBSEncrypted() bool
}

type Environment interface {
	Tarmak() Tarmak
	Location() string // this returns the location of the environment (e.g. the region)
	Variables() map[string]interface{}
	Provider() Provider
	// Verify the cluster (these contain more expensive calls like AWS calls
	Verify() error
	// Validate the cluster (these contain less expensive local calls)
	Validate() error
	Name() string
	HubName() string
	Clusters() []Cluster
	Cluster(name string) (cluster Cluster, err error)
	SSHPrivateKeyPath() string
	SSHPrivateKey() (signer interface{})
	Log() *logrus.Entry
	Parameters() map[string]string
	Config() *tarmakv1alpha1.Environment
	Type() string
	WingTunnel() Tunnel
	WingClientset() (*wingclient.Clientset, Tunnel, error)

	// get the absolute config path to the environment's config folder
	ConfigPath() string

	// this verifies if the connection to the bastion instance is working
	VerifyBastionAvailable() error

	// return the cluster which is the hub
	Hub() Cluster

	// return the vault for the environment
	Vault() Vault
}

type Provider interface {
	Cloud() string
	Name() string
	Parameters() map[string]string
	Region() string
	// Verify the cluster (these contain more expensive calls like AWS calls)
	Verify() error
	// Validate the cluster (these contain less expensive local calls)
	Validate() error
	Reset() // reset all caches within the provider
	RemoteStateBucketName() string
	RemoteStateBucketAvailable() (bool, error)
	RemoteState(namespace, clusterName, stackName string) string
	PublicZone() string
	Environment() ([]string, error)
	Variables() map[string]interface{}
	QueryImages(tags map[string]string) ([]*tarmakv1alpha1.Image, error)
	DefaultImage(version string) (*tarmakv1alpha1.Image, error)
	VaultKV() (kv.Service, error)
	VaultKVWithParams(kmsKeyID, unsealKeyName string) (kv.Service, error)
	ListHosts(Cluster) ([]Host, error)
	InstanceType(string) (string, error)
	VolumeType(string) (string, error)
	String() string
	AskEnvironmentLocation(Initialize) (string, error)
	AskInstancePoolZones(Initialize) (zones []string, err error)
	UploadConfiguration(Cluster, io.ReadSeeker, string) error
	EnsureRemoteResources() error
	LegacyPuppetTFName() string
	// Remove provider
	Remove() error
}

type Tarmak interface {
	Variables() map[string]interface{}
	Log() *logrus.Entry
	RootPath() (string, error)

	// get the absolute config path to tarmak's config folder
	ConfigPath() string

	Cluster() Cluster
	ClusterFlags() tarmakv1alpha1.ClusterFlags
	Environments() []Environment
	Environment() Environment
	Providers() []Provider
	Provider() Provider
	Terraform() Terraform
	Packer() Packer
	Puppet() Puppet
	Config() Config
	SSH() SSH
	Version() string
	HomeDirExpand(in string) (string, error)
	HomeDir() string
	KeepContainers() bool
	CancellationContext() CancellationContext
	Cleanup()

	// get a provider by name
	ProviderByName(string) (Provider, error)
	// get an environment by name
	EnvironmentByName(string) (Environment, error)
	EnsureRemoteResources() error
}

type Config interface {
	Cluster(environment string, name string) (cluster *clusterv1alpha1.Cluster, err error)
	Clusters(environment string) (clusters []*clusterv1alpha1.Cluster)
	AppendCluster(cluster *clusterv1alpha1.Cluster) error
	UniqueClusterName(environment, name string) error
	Provider(name string) (provider *tarmakv1alpha1.Provider, err error)
	Providers() (providers []*tarmakv1alpha1.Provider)
	AppendProvider(prov *tarmakv1alpha1.Provider) error
	UniqueProviderName(name string) error
	ValidName(name, regex string) error
	ReadConfig() (*tarmakv1alpha1.Config, error)
	Environment(name string) (environment *tarmakv1alpha1.Environment, err error)
	Environments() (environments []*tarmakv1alpha1.Environment)
	AppendEnvironment(*tarmakv1alpha1.Environment) error
	UniqueEnvironmentName(name string) error
	// currently selected <env name>-<cluster name>
	CurrentCluster() (string, error)
	// currently selected cluster name
	CurrentClusterName() (string, error)
	// currently selected env name
	CurrentEnvironmentName() (string, error)
	// remove environment
	RemoveEnvironment(environment string) error
	Contact() string
	Project() string
	WingDevMode() bool
	SetCurrentCluster(string) error
	IgnoreMissingPublicKeyTags() bool
}

type Packer interface {
	IDs(encrypted bool) (map[string]string, error)
	List() ([]*tarmakv1alpha1.Image, error)
	Build(imageNames []string) error
}

type Terraform interface {
	Output(cluster Cluster) (map[string]interface{}, error)
	Prepare(cluster Cluster) error
}

type SSH interface {
	WriteConfig(Cluster) error
	PassThrough([]string) error
	Tunnel(destination, destinationPort, localPort string, daemonize bool) Tunnel
	Execute(host string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) (returnCode int, err error)
	Validate() error
	Cleanup()
}

type Logs interface {
	Aggregate(group string, flags tarmakv1alpha1.ClusterLogsFlags) error
}

type Tunnel interface {
	Start() error
	Stop()
	Port() string
	BindAddress() string
	Done() <-chan struct{}
}

type VaultTunnel interface {
	Tunnel
	VaultClient() *vault.Client
}

type Host interface {
	ID() string
	Hostname() string
	User() string
	Roles() []string
	SSHConfig(strictChecking string) string
	Parameters() map[string]string
	SSHHostPublicKeys() ([]ssh.PublicKey, error)
	Aliases() []string
}

type Puppet interface {
	TarGz(io.Writer) error
	Initialize(packerbuild bool) error
}

type Kubectl interface {
}

type Vault interface {
	Tunnel() (VaultTunnel, error)
	RootToken() (string, error)
	TunnelFromFQDNs(vaultInternalFQDNs []string, vaultCA string) (VaultTunnel, error)
	VerifyInitFromFQDNs(instances []string, vaultCA, vaultKMSKeyID, vaultUnsealKeyName string) error
}

type InstancePool interface {
	AmazonAdditionalIAMPolicies() []string
	Config() *clusterv1alpha1.InstancePool
	TFName() string
	Name() string
	Image() string
	Role() *role.Role
	Volumes() []Volume
	RootVolume() Volume
	Zones() []string
	Validate() error
	MinCount() int
	MaxCount() int
	InstanceType() string
	Labels() (string, error)
	Taints() (string, error)
}

type Volume interface {
	Name() string
	Size() int
	Type() string
	Device() string
}

type Initialize interface {
	Input() *input.Input
	AskProjectName() (string, error)
	AskContact() (string, error)
	Config() Config
	Tarmak() Tarmak
	CurrentProvider() Provider
	CurrentEnvironment() Environment
}

type CancellationContext interface {
	Done() <-chan struct{}
	Err() error
	Signal() os.Signal
	WaitOrCancel(f func() error)
	WaitOrCancelReturnCode(f func() (int, error))
	TryOrCancel(done <-chan struct{}) context.Context
}
