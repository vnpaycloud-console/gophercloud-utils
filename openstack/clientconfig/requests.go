package clientconfig

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/vnpaycloud-console/gophercloud-utils/v2/env"
	"github.com/vnpaycloud-console/gophercloud-utils/v2/gnocchi"
	"github.com/vnpaycloud-console/gophercloud-utils/v2/internal"
	"github.com/vnpaycloud-console/gophercloud/v2"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack"

	"github.com/gofrs/uuid/v5"

	yaml "gopkg.in/yaml.v3"
)

// AuthType respresents a valid method of authentication.
type AuthType string

const (
	// AuthPassword defines an unknown version of the password
	AuthPassword AuthType = "password"
	// AuthToken defined an unknown version of the token
	AuthToken AuthType = "token"

	// AuthV2Password defines version 2 of the password
	AuthV2Password AuthType = "v2password"
	// AuthV2Token defines version 2 of the token
	AuthV2Token AuthType = "v2token"

	// AuthV3Password defines version 3 of the password
	AuthV3Password AuthType = "v3password"
	// AuthV3Token defines version 3 of the token
	AuthV3Token AuthType = "v3token"

	// AuthV3ApplicationCredential defines version 3 of the application credential
	AuthV3ApplicationCredential AuthType = "v3applicationcredential"
)

// ClientOpts represents options to customize the way a client is
// configured.
type ClientOpts struct {
	// Cloud is the cloud entry in clouds.yaml to use.
	Cloud string

	// EnvPrefix allows a custom environment variable prefix to be used.
	EnvPrefix string

	// AuthType specifies the type of authentication to use.
	// By default, this is "password".
	AuthType AuthType

	// AuthInfo defines the authentication information needed to
	// authenticate to a cloud when clouds.yaml isn't used.
	AuthInfo *AuthInfo

	// RegionName is the region to create a Service Client in.
	// This will override a region in clouds.yaml or can be used
	// when authenticating directly with AuthInfo.
	RegionName string

	// EndpointType specifies whether to use the public, internal, or
	// admin endpoint of a service.
	EndpointType string

	// HTTPClient provides the ability customize the ProviderClient's
	// internal HTTP client.
	HTTPClient *http.Client

	// YAMLOpts provides the ability to pass a customized set
	// of options and methods for loading the YAML file.
	// It takes a YAMLOptsBuilder interface that is defined
	// in this file. This is optional and the default behavior
	// is to call the local LoadCloudsYAML functions defined
	// in this file.
	YAMLOpts YAMLOptsBuilder
}

// YAMLOptsBuilder defines an interface for customization when
// loading a clouds.yaml file.
type YAMLOptsBuilder interface {
	LoadCloudsYAML() (map[string]Cloud, error)
	LoadSecureCloudsYAML() (map[string]Cloud, error)
	LoadPublicCloudsYAML() (map[string]Cloud, error)
}

// YAMLOpts represents options and methods to load a clouds.yaml file.
type YAMLOpts struct {
	// By default, no options are specified.
}

// LoadCloudsYAML defines how to load a clouds.yaml file.
// By default, this calls the local LoadCloudsYAML function.
func (opts YAMLOpts) LoadCloudsYAML() (map[string]Cloud, error) {
	return LoadCloudsYAML()
}

// LoadSecureCloudsYAML defines how to load a secure.yaml file.
// By default, this calls the local LoadSecureCloudsYAML function.
func (opts YAMLOpts) LoadSecureCloudsYAML() (map[string]Cloud, error) {
	return LoadSecureCloudsYAML()
}

// LoadPublicCloudsYAML defines how to load a public-secure.yaml file.
// By default, this calls the local LoadPublicCloudsYAML function.
func (opts YAMLOpts) LoadPublicCloudsYAML() (map[string]Cloud, error) {
	return LoadPublicCloudsYAML()
}

// LoadCloudsYAML will load a clouds.yaml file and return the full config.
// This is called by the YAMLOpts method. Calling this function directly
// is supported for now but has only been retained for backwards
// compatibility from before YAMLOpts was defined. This may be removed in
// the future.
func LoadCloudsYAML() (map[string]Cloud, error) {
	_, content, err := FindAndReadCloudsYAML()
	if err != nil {
		return nil, err
	}

	var clouds Clouds
	err = yaml.Unmarshal(content, &clouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return clouds.Clouds, nil
}

// LoadSecureCloudsYAML will load a secure.yaml file and return the full config.
// This is called by the YAMLOpts method. Calling this function directly
// is supported for now but has only been retained for backwards
// compatibility from before YAMLOpts was defined. This may be removed in
// the future.
func LoadSecureCloudsYAML() (map[string]Cloud, error) {
	var secureClouds Clouds

	_, content, err := FindAndReadSecureCloudsYAML()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// secure.yaml is optional so just ignore read error
			return secureClouds.Clouds, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(content, &secureClouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return secureClouds.Clouds, nil
}

// LoadPublicCloudsYAML will load a public-clouds.yaml file and return the full config.
// This is called by the YAMLOpts method. Calling this function directly
// is supported for now but has only been retained for backwards
// compatibility from before YAMLOpts was defined. This may be removed in
// the future.
func LoadPublicCloudsYAML() (map[string]Cloud, error) {
	var publicClouds PublicClouds

	_, content, err := FindAndReadPublicCloudsYAML()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// clouds-public.yaml is optional so just ignore read error
			return publicClouds.Clouds, nil
		}

		return nil, err
	}

	err = yaml.Unmarshal(content, &publicClouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return publicClouds.Clouds, nil
}

// GetCloudFromYAML will return a cloud entry from a clouds.yaml file.
func GetCloudFromYAML(opts *ClientOpts) (*Cloud, error) {
	if opts == nil {
		opts = new(ClientOpts)
	}

	if opts.YAMLOpts == nil {
		opts.YAMLOpts = new(YAMLOpts)
	}

	yamlOpts := opts.YAMLOpts

	clouds, err := yamlOpts.LoadCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load clouds.yaml: %w", err)
	}

	// Determine which cloud to use.
	// First see if a cloud name was explicitly set in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	} else {
		// If not, see if a cloud name was specified as an environment variable.
		envPrefix := "OS_"
		if opts.EnvPrefix != "" {
			envPrefix = opts.EnvPrefix
		}

		if v := env.Getenv(envPrefix + "CLOUD"); v != "" {
			cloudName = v
		}
	}

	var cloud *Cloud
	if cloudName != "" {
		v, ok := clouds[cloudName]
		if !ok {
			return nil, fmt.Errorf("cloud %s does not exist in clouds.yaml", cloudName)
		}
		cloud = &v
	}

	// If a cloud was not specified, and clouds only contains
	// a single entry, use that entry.
	if cloudName == "" && len(clouds) == 1 {
		for _, v := range clouds {
			cloud = &v
		}
	}

	if cloud != nil {
		// A profile points to a public cloud entry.
		// If one was specified, load a list of public clouds
		// and then merge the information with the current cloud data.
		profileName := defaultIfEmpty(cloud.Profile, cloud.Cloud)

		if profileName != "" {
			publicClouds, err := yamlOpts.LoadPublicCloudsYAML()
			if err != nil {
				return nil, fmt.Errorf("unable to load clouds-public.yaml: %w", err)
			}

			publicCloud, ok := publicClouds[profileName]
			if !ok {
				return nil, fmt.Errorf("cloud %s does not exist in clouds-public.yaml", profileName)
			}

			cloud, err = mergeClouds(cloud, publicCloud)
			if err != nil {
				return nil, fmt.Errorf("Could not merge information from clouds.yaml and clouds-public.yaml for cloud %s", profileName)
			}
		}
	}

	// Next, load a secure clouds file and see if a cloud entry
	// can be found or merged.
	secureClouds, err := yamlOpts.LoadSecureCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load secure.yaml: %w", err)
	}

	if secureClouds != nil {
		// If no entry was found in clouds.yaml, no cloud name was specified,
		// and only one secureCloud entry exists, use that as the cloud entry.
		if cloud == nil && cloudName == "" && len(secureClouds) == 1 {
			for _, v := range secureClouds {
				cloud = &v
			}
		}

		// Otherwise, see if the provided cloud name exists in the secure yaml file.
		secureCloud, ok := secureClouds[cloudName]
		if !ok && cloud == nil {
			// cloud == nil serves two purposes here:
			// if no entry in clouds.yaml was found and
			// if a single-entry secureCloud wasn't used.
			// At this point, no entry could be determined at all.
			return nil, fmt.Errorf("Could not find cloud %s", cloudName)
		}

		// If secureCloud has content and it differs from the cloud entry,
		// merge the two together.
		if !reflect.DeepEqual((Cloud{}), secureCloud) && !reflect.DeepEqual(cloud, secureCloud) {
			cloud, err = mergeClouds(secureCloud, cloud)
			if err != nil {
				return nil, fmt.Errorf("unable to merge information from clouds.yaml and secure.yaml")
			}
		}
	}

	// As an extra precaution, do one final check to see if cloud is nil.
	// We shouldn't reach this point, though.
	if cloud == nil {
		return nil, fmt.Errorf("Could not find cloud %s", cloudName)
	}

	// Default is to verify SSL API requests
	if cloud.Verify == nil {
		iTrue := true
		cloud.Verify = &iTrue
	}

	// merging per-region value overrides
	if opts.RegionName != "" {
		for _, v := range cloud.Regions {
			if opts.RegionName == v.Name {
				cloud, err = mergeClouds(v.Values, cloud)
				break
			}
		}
	}

	// TODO: this is where reading vendor files should go be considered when not found in
	// clouds-public.yml
	// https://github.com/openstack/openstacksdk/tree/master/openstack/config/vendors

	// Both Interface and EndpointType are valid settings in clouds.yaml,
	// but we want to standardize on EndpointType for simplicity.
	//
	// If only Interface was set, we copy that to EndpointType to use as the setting.
	// But in all other cases, EndpointType is used and Interface is cleared.
	if cloud.Interface != "" && cloud.EndpointType == "" {
		cloud.EndpointType = cloud.Interface
	}

	cloud.Interface = ""

	return cloud, err
}

// AuthOptions creates a gophercloud.AuthOptions structure with the
// settings found in a specific cloud entry of a clouds.yaml file or
// based on authentication settings given in ClientOpts.
//
// This attempts to be a single point of entry for all OpenStack authentication.
//
// See http://docs.openstack.org/developer/os-client-config and
// https://github.com/openstack/os-client-config/blob/master/os_client_config/config.py.
func AuthOptions(opts *ClientOpts) (*gophercloud.AuthOptions, error) {
	cloud := new(Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	} else {
		// If not, see if a cloud name was specified as an environment
		// variable.
		envPrefix := "OS_"
		if opts.EnvPrefix != "" {
			envPrefix = opts.EnvPrefix
		}

		if v := env.Getenv(envPrefix + "CLOUD"); v != "" {
			cloudName = v
		}
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// If cloud.AuthInfo is nil, then no cloud was specified.
	if cloud.AuthInfo == nil {
		// If opts.AuthInfo is not nil, then try using the auth settings from it.
		if opts.AuthInfo != nil {
			cloud.AuthInfo = opts.AuthInfo
		}

		// If cloud.AuthInfo is still nil, then set it to an empty Auth struct
		// and rely on environment variables to do the authentication.
		if cloud.AuthInfo == nil {
			cloud.AuthInfo = new(AuthInfo)
		}
	}

	identityAPI := determineIdentityAPI(cloud, opts)
	switch identityAPI {
	case "2.0", "2":
		return v2auth(cloud, opts)
	case "3":
		return v3auth(cloud, opts)
	}

	return nil, fmt.Errorf("Unable to build AuthOptions")
}

func determineIdentityAPI(cloud *Cloud, opts *ClientOpts) string {
	var identityAPI string
	if cloud.IdentityAPIVersion != "" {
		identityAPI = cloud.IdentityAPIVersion
	}

	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := env.Getenv(envPrefix + "IDENTITY_API_VERSION"); v != "" {
		identityAPI = v
	}

	if identityAPI == "" {
		if cloud.AuthInfo != nil {
			if strings.Contains(cloud.AuthInfo.AuthURL, "v2.0") {
				identityAPI = "2.0"
			}

			if strings.Contains(cloud.AuthInfo.AuthURL, "v3") {
				identityAPI = "3"
			}
		}
	}

	if identityAPI == "" {
		switch cloud.AuthType {
		case AuthV2Password:
			identityAPI = "2.0"
		case AuthV2Token:
			identityAPI = "2.0"
		case AuthV3Password:
			identityAPI = "3"
		case AuthV3Token:
			identityAPI = "3"
		case AuthV3ApplicationCredential:
			identityAPI = "3"
		}
	}

	// If an Identity API version could not be determined,
	// default to v3.
	if identityAPI == "" {
		identityAPI = "3"
	}

	return identityAPI
}

// v2auth creates a v2-compatible gophercloud.AuthOptions struct.
func v2auth(cloud *Cloud, opts *ClientOpts) (*gophercloud.AuthOptions, error) {
	// Environment variable overrides.
	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if cloud.AuthInfo.AuthURL == "" {
		if v := env.Getenv(envPrefix + "AUTH_URL"); v != "" {
			cloud.AuthInfo.AuthURL = v
		}
	}

	if cloud.AuthInfo.Token == "" {
		if v := env.Getenv(envPrefix + "TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}

		if v := env.Getenv(envPrefix + "AUTH_TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}
	}

	if cloud.AuthInfo.Username == "" {
		if v := env.Getenv(envPrefix + "USERNAME"); v != "" {
			cloud.AuthInfo.Username = v
		}
	}

	if cloud.AuthInfo.Password == "" {
		if v := env.Getenv(envPrefix + "PASSWORD"); v != "" {
			cloud.AuthInfo.Password = v
		}
	}

	if cloud.AuthInfo.ProjectID == "" {
		if v := env.Getenv(envPrefix + "TENANT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}

		if v := env.Getenv(envPrefix + "PROJECT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}
	}

	if cloud.AuthInfo.ProjectName == "" {
		if v := env.Getenv(envPrefix + "TENANT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}

		if v := env.Getenv(envPrefix + "PROJECT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}
	}

	ao := &gophercloud.AuthOptions{
		IdentityEndpoint: cloud.AuthInfo.AuthURL,
		TokenID:          cloud.AuthInfo.Token,
		Username:         cloud.AuthInfo.Username,
		Password:         cloud.AuthInfo.Password,
		TenantID:         cloud.AuthInfo.ProjectID,
		TenantName:       cloud.AuthInfo.ProjectName,
		AllowReauth:      cloud.AuthInfo.AllowReauth,
	}

	return ao, nil
}

// v3auth creates a v3-compatible gophercloud.AuthOptions struct.
func v3auth(cloud *Cloud, opts *ClientOpts) (*gophercloud.AuthOptions, error) {
	// Environment variable overrides.
	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if cloud.AuthInfo.AuthURL == "" {
		if v := env.Getenv(envPrefix + "AUTH_URL"); v != "" {
			cloud.AuthInfo.AuthURL = v
		}
	}

	if cloud.AuthInfo.Token == "" {
		if v := env.Getenv(envPrefix + "TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}

		if v := env.Getenv(envPrefix + "AUTH_TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}
	}

	if cloud.AuthInfo.Username == "" {
		if v := env.Getenv(envPrefix + "USERNAME"); v != "" {
			cloud.AuthInfo.Username = v
		}
	}

	if cloud.AuthInfo.UserID == "" {
		if v := env.Getenv(envPrefix + "USER_ID"); v != "" {
			cloud.AuthInfo.UserID = v
		}
	}

	if cloud.AuthInfo.Password == "" {
		if v := env.Getenv(envPrefix + "PASSWORD"); v != "" {
			cloud.AuthInfo.Password = v
		}
	}

	if cloud.AuthInfo.ProjectID == "" {
		if v := env.Getenv(envPrefix + "TENANT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}

		if v := env.Getenv(envPrefix + "PROJECT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}
	}

	if cloud.AuthInfo.ProjectName == "" {
		if v := env.Getenv(envPrefix + "TENANT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}

		if v := env.Getenv(envPrefix + "PROJECT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}
	}

	if cloud.AuthInfo.DomainID == "" {
		if v := env.Getenv(envPrefix + "DOMAIN_ID"); v != "" {
			cloud.AuthInfo.DomainID = v
		}
	}

	if cloud.AuthInfo.DomainName == "" {
		if v := env.Getenv(envPrefix + "DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.DomainName = v
		}
	}

	if cloud.AuthInfo.DefaultDomain == "" {
		if v := env.Getenv(envPrefix + "DEFAULT_DOMAIN"); v != "" {
			cloud.AuthInfo.DefaultDomain = v
		}
	}

	if cloud.AuthInfo.ProjectDomainID == "" {
		if v := env.Getenv(envPrefix + "PROJECT_DOMAIN_ID"); v != "" {
			cloud.AuthInfo.ProjectDomainID = v
		}
	}

	if cloud.AuthInfo.ProjectDomainName == "" {
		if v := env.Getenv(envPrefix + "PROJECT_DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.ProjectDomainName = v
		}
	}

	if cloud.AuthInfo.UserDomainID == "" {
		if v := env.Getenv(envPrefix + "USER_DOMAIN_ID"); v != "" {
			cloud.AuthInfo.UserDomainID = v
		}
	}

	if cloud.AuthInfo.UserDomainName == "" {
		if v := env.Getenv(envPrefix + "USER_DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.UserDomainName = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialID == "" {
		if v := env.Getenv(envPrefix + "APPLICATION_CREDENTIAL_ID"); v != "" {
			cloud.AuthInfo.ApplicationCredentialID = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialName == "" {
		if v := env.Getenv(envPrefix + "APPLICATION_CREDENTIAL_NAME"); v != "" {
			cloud.AuthInfo.ApplicationCredentialName = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialSecret == "" {
		if v := env.Getenv(envPrefix + "APPLICATION_CREDENTIAL_SECRET"); v != "" {
			cloud.AuthInfo.ApplicationCredentialSecret = v
		}
	}

	if cloud.AuthInfo.SystemScope == "" {
		if v := env.Getenv(envPrefix + "SYSTEM_SCOPE"); v != "" {
			cloud.AuthInfo.SystemScope = v
		}
	}

	// Build a scope and try to do it correctly.
	// https://github.com/openstack/os-client-config/blob/master/os_client_config/config.py#L595
	scope := new(gophercloud.AuthScope)

	// Application credentials don't support scope
	if isApplicationCredential(cloud.AuthInfo) {
		// If Domain* is set, but UserDomain* or ProjectDomain* aren't,
		// then use Domain* as the default setting.
		cloud = setDomainIfNeeded(cloud)
	} else {
		if !isProjectScoped(cloud.AuthInfo) {
			if cloud.AuthInfo.DomainID != "" {
				scope.DomainID = cloud.AuthInfo.DomainID
			} else if cloud.AuthInfo.DomainName != "" {
				scope.DomainName = cloud.AuthInfo.DomainName
			}
			if cloud.AuthInfo.SystemScope != "" {
				scope.System = true
			}
		} else {
			// If Domain* is set, but UserDomain* or ProjectDomain* aren't,
			// then use Domain* as the default setting.
			cloud = setDomainIfNeeded(cloud)

			if cloud.AuthInfo.ProjectID != "" {
				scope.ProjectID = cloud.AuthInfo.ProjectID
			} else {
				scope.ProjectName = cloud.AuthInfo.ProjectName
				scope.DomainID = cloud.AuthInfo.ProjectDomainID
				scope.DomainName = cloud.AuthInfo.ProjectDomainName
			}
		}
	}

	ao := &gophercloud.AuthOptions{
		Scope:                       scope,
		IdentityEndpoint:            cloud.AuthInfo.AuthURL,
		TokenID:                     cloud.AuthInfo.Token,
		Username:                    cloud.AuthInfo.Username,
		UserID:                      cloud.AuthInfo.UserID,
		Password:                    cloud.AuthInfo.Password,
		TenantID:                    cloud.AuthInfo.ProjectID,
		TenantName:                  cloud.AuthInfo.ProjectName,
		DomainID:                    cloud.AuthInfo.UserDomainID,
		DomainName:                  cloud.AuthInfo.UserDomainName,
		ApplicationCredentialID:     cloud.AuthInfo.ApplicationCredentialID,
		ApplicationCredentialName:   cloud.AuthInfo.ApplicationCredentialName,
		ApplicationCredentialSecret: cloud.AuthInfo.ApplicationCredentialSecret,
		AllowReauth:                 cloud.AuthInfo.AllowReauth,
	}

	// If an auth_type of "token" was specified, then make sure
	// Gophercloud properly authenticates with a token. This involves
	// unsetting a few other auth options. The reason this is done
	// here is to wait until all auth settings (both in clouds.yaml
	// and via environment variables) are set and then unset them.
	if strings.Contains(string(cloud.AuthType), "token") || ao.TokenID != "" {
		ao.Username = ""
		ao.Password = ""
		ao.UserID = ""
		ao.DomainID = ""
		ao.DomainName = ""
	}

	// Check for absolute minimum requirements.
	if ao.IdentityEndpoint == "" {
		err := gophercloud.ErrMissingInput{Argument: "auth_url"}
		return nil, err
	}

	return ao, nil
}

// AuthenticatedClient is a convenience function to get a new provider client
// based on a clouds.yaml entry.
func AuthenticatedClient(ctx context.Context, opts *ClientOpts) (*gophercloud.ProviderClient, error) {
	ao, err := AuthOptions(opts)
	if err != nil {
		return nil, err
	}

	return openstack.AuthenticatedClient(ctx, *ao)
}

// NewServiceClient is a convenience function to get a new service client.
func NewServiceClient(ctx context.Context, service string, opts *ClientOpts) (*gophercloud.ServiceClient, error) {
	cloud := new(Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := env.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// Check if a custom CA cert was provided.
	// First, check if the CACERT environment variable is set.
	var caCertPath string
	if v := env.Getenv(envPrefix + "CACERT"); v != "" {
		caCertPath = v
	}
	// Next, check if the cloud entry sets a CA cert.
	if v := cloud.CACertFile; v != "" {
		caCertPath = v
	}

	// Check if a custom client cert was provided.
	// First, check if the CERT environment variable is set.
	var clientCertPath string
	if v := env.Getenv(envPrefix + "CERT"); v != "" {
		clientCertPath = v
	}
	// Next, check if the cloud entry sets a client cert.
	if v := cloud.ClientCertFile; v != "" {
		clientCertPath = v
	}

	// Check if a custom client key was provided.
	// First, check if the KEY environment variable is set.
	var clientKeyPath string
	if v := env.Getenv(envPrefix + "KEY"); v != "" {
		clientKeyPath = v
	}
	// Next, check if the cloud entry sets a client key.
	if v := cloud.ClientKeyFile; v != "" {
		clientKeyPath = v
	}

	// Define whether or not SSL API requests should be verified.
	var insecurePtr *bool
	if cloud.Verify != nil {
		// Here we take the boolean pointer negation.
		insecure := !*cloud.Verify
		insecurePtr = &insecure
	}

	tlsConfig, err := internal.PrepareTLSConfig(caCertPath, clientCertPath, clientKeyPath, insecurePtr)
	if err != nil {
		return nil, err
	}

	// Get a Provider Client
	ao, err := AuthOptions(opts)
	if err != nil {
		return nil, err
	}
	pClient, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	// If an HTTPClient was specified, use it.
	if opts.HTTPClient != nil {
		pClient.HTTPClient = *opts.HTTPClient
	} else {
		// Otherwise create a new HTTP client with the generated TLS config.
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = tlsConfig
		pClient.HTTPClient = http.Client{Transport: transport}
	}

	err = openstack.Authenticate(ctx, pClient, *ao)
	if err != nil {
		return nil, err
	}

	// Determine the region to use.
	// First, check if the REGION_NAME environment variable is set.
	var region string
	if v := env.Getenv(envPrefix + "REGION_NAME"); v != "" {
		region = v
	}

	// Next, check if the cloud entry sets a region.
	if v := cloud.RegionName; v != "" {
		region = v
	}

	// Finally, see if one was specified in the ClientOpts.
	// If so, this takes precedence.
	if v := opts.RegionName; v != "" {
		region = v
	}

	// Determine the endpoint type to use.
	// First, check if the OS_INTERFACE environment variable is set.
	var endpointType string
	if v := env.Getenv(envPrefix + "INTERFACE"); v != "" {
		endpointType = v
	}

	// Next, check if the cloud entry sets an endpoint type.
	if v := cloud.EndpointType; v != "" {
		endpointType = v
	}

	// Finally, see if one was specified in the ClientOpts.
	// If so, this takes precedence.
	if v := opts.EndpointType; v != "" {
		endpointType = v
	}

	eo := gophercloud.EndpointOpts{
		Region:       region,
		Availability: GetEndpointType(endpointType),
	}

	switch service {
	case "baremetal":
		return openstack.NewBareMetalV1(pClient, eo)
	case "baremetal-introspection":
		return openstack.NewBareMetalIntrospectionV1(pClient, eo)
	case "compute":
		return openstack.NewComputeV2(pClient, eo)
	case "container":
		return openstack.NewContainerV1(pClient, eo)
	case "container-infra":
		return openstack.NewContainerInfraV1(pClient, eo)
	case "database":
		return openstack.NewDBV1(pClient, eo)
	case "dns":
		return openstack.NewDNSV2(pClient, eo)
	case "gnocchi":
		return gnocchi.NewGnocchiV1(pClient, eo)
	case "identity":
		identityVersion := "3"
		if v := cloud.IdentityAPIVersion; v != "" {
			identityVersion = v
		}

		switch identityVersion {
		case "v2", "2", "2.0":
			return openstack.NewIdentityV2(pClient, eo)
		case "v3", "3":
			return openstack.NewIdentityV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid identity API version")
		}
	case "image":
		return openstack.NewImageV2(pClient, eo)
	case "key-manager":
		return openstack.NewKeyManagerV1(pClient, eo)
	case "load-balancer":
		return openstack.NewLoadBalancerV2(pClient, eo)
	case "messaging":
		clientID, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("failed to generate UUID: %w", err)
		}
		return openstack.NewMessagingV2(pClient, clientID.String(), eo)
	case "network":
		return openstack.NewNetworkV2(pClient, eo)
	case "object-store":
		return openstack.NewObjectStorageV1(pClient, eo)
	case "orchestration":
		return openstack.NewOrchestrationV1(pClient, eo)
	case "placement":
		return openstack.NewPlacementV1(pClient, eo)
	case "sharev2":
		return openstack.NewSharedFileSystemV2(pClient, eo)
	case "volume":
		volumeVersion := "3"
		if v := cloud.VolumeAPIVersion; v != "" {
			volumeVersion = v
		}

		switch volumeVersion {
		case "v1", "1":
			return openstack.NewBlockStorageV1(pClient, eo)
		case "v2", "2":
			return openstack.NewBlockStorageV2(pClient, eo)
		case "v3", "3":
			return openstack.NewBlockStorageV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid volume API version")
		}
	case "workflowv2":
		return openstack.NewWorkflowV2(pClient, eo)
	}

	return nil, fmt.Errorf("unable to create a service client for %s", service)
}

// isProjectScoped determines if an auth struct is project scoped.
func isProjectScoped(authInfo *AuthInfo) bool {
	if authInfo.ProjectID == "" && authInfo.ProjectName == "" {
		return false
	}

	return true
}

// setDomainIfNeeded will set a DomainID and DomainName
// to ProjectDomain* and UserDomain* if not already set.
func setDomainIfNeeded(cloud *Cloud) *Cloud {
	if cloud.AuthInfo.DomainID != "" {
		if cloud.AuthInfo.UserDomainID == "" {
			cloud.AuthInfo.UserDomainID = cloud.AuthInfo.DomainID
		}

		if cloud.AuthInfo.ProjectDomainID == "" {
			cloud.AuthInfo.ProjectDomainID = cloud.AuthInfo.DomainID
		}

		cloud.AuthInfo.DomainID = ""
	}

	if cloud.AuthInfo.DomainName != "" {
		if cloud.AuthInfo.UserDomainName == "" {
			cloud.AuthInfo.UserDomainName = cloud.AuthInfo.DomainName
		}

		if cloud.AuthInfo.ProjectDomainName == "" {
			cloud.AuthInfo.ProjectDomainName = cloud.AuthInfo.DomainName
		}

		cloud.AuthInfo.DomainName = ""
	}

	// If Domain fields are still not set, and if DefaultDomain has a value,
	// set UserDomainID and ProjectDomainID to DefaultDomain.
	// https://github.com/openstack/osc-lib/blob/86129e6f88289ef14bfaa3f7c9cdfbea8d9fc944/osc_lib/cli/client_config.py#L117-L146
	if cloud.AuthInfo.DefaultDomain != "" {
		if cloud.AuthInfo.UserDomainName == "" && cloud.AuthInfo.UserDomainID == "" {
			cloud.AuthInfo.UserDomainID = cloud.AuthInfo.DefaultDomain
		}

		if cloud.AuthInfo.ProjectDomainName == "" && cloud.AuthInfo.ProjectDomainID == "" {
			cloud.AuthInfo.ProjectDomainID = cloud.AuthInfo.DefaultDomain
		}
	}

	return cloud
}

// isApplicationCredential determines if an application credential is used to auth.
func isApplicationCredential(authInfo *AuthInfo) bool {
	if authInfo.ApplicationCredentialID == "" && authInfo.ApplicationCredentialName == "" && authInfo.ApplicationCredentialSecret == "" {
		return false
	}
	return true
}
