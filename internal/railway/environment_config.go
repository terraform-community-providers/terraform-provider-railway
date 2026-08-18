package railway

type EnvironmentConfig struct {
	Buckets  map[string]BucketConfig  `json:"buckets,omitempty"`
	Services map[string]ServiceConfig `json:"services,omitempty"`
	Volumes  map[string]VolumeConfig  `json:"volumes,omitempty"`
}

type BucketConfig struct {
	Region    string `json:"region,omitempty"`
	IsCreated bool   `json:"isCreated"`
	IsDeleted bool   `json:"isDeleted"`
}

type VolumeConfig struct {
	SizeMB    int64 `json:"sizeMB,omitempty"`
	IsCreated bool  `json:"isCreated,omitempty"`
}

type ServiceConfig struct {
	ConfigFile *string                    `json:"configFile"`
	Deploy     *ServiceDeployConfig       `json:"deploy,omitempty"`
	IsCreated  bool                       `json:"isCreated,omitempty"`
	IsDeleted  bool                       `json:"isDeleted,omitempty"`
	Source     *ServiceSourceConfig       `json:"source,omitempty"`
	Variables  map[string]*VariableConfig `json:"variables,omitempty"`
}

type ServiceDeployConfig struct {
	CronSchedule        *string                           `json:"cronSchedule"`
	HealthcheckPath     *string                           `json:"healthcheckPath"`
	HealthcheckTimeout  *int64                            `json:"healthcheckTimeout"`
	LimitOverride       *ServiceLimitOverrideConfig       `json:"limitOverride,omitempty"`
	MultiRegionConfig   map[string]*ServiceRegionConfig   `json:"multiRegionConfig"`
	RegistryCredentials *ServiceRegistryCredentialsConfig `json:"registryCredentials"`
	StartCommand        *string                           `json:"startCommand"`
}

type ServiceLimitOverrideConfig struct {
	Containers *ServiceContainerLimitsConfig `json:"containers"`
}

type ServiceContainerLimitsConfig struct {
	CPU         *float64 `json:"cpu"`
	MemoryBytes *float64 `json:"memoryBytes"`
}

type ServiceRegionConfig struct {
	NumReplicas int64 `json:"numReplicas"`
}

type ServiceSourceConfig struct {
	Branch        *string `json:"branch"`
	Image         *string `json:"image"`
	Repo          *string `json:"repo"`
	RootDirectory *string `json:"rootDirectory"`
}

type ServiceRegistryCredentialsConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type VariableConfig struct {
	Value    string `json:"value"`
	IsSealed bool   `json:"isSealed"`
}

func SealedVariablePatch(serviceID string, name string, value string) EnvironmentConfig {
	return variablePatch(serviceID, name, &VariableConfig{
		Value:    value,
		IsSealed: true,
	})
}

func DeleteVariablePatch(serviceID string, name string) EnvironmentConfig {
	return variablePatch(serviceID, name, nil)
}

func CreateBucketPatch(bucketID string, region string) EnvironmentConfig {
	return EnvironmentConfig{
		Buckets: map[string]BucketConfig{
			bucketID: {
				Region:    region,
				IsCreated: true,
			},
		},
	}
}

func DeleteBucketPatch(bucketID string) EnvironmentConfig {
	return EnvironmentConfig{
		Buckets: map[string]BucketConfig{
			bucketID: {
				IsDeleted: true,
			},
		},
	}
}

func CreateVolumePatch(volumeID string, sizeMB int64) EnvironmentConfig {
	return EnvironmentConfig{
		Volumes: map[string]VolumeConfig{
			volumeID: {
				SizeMB:    sizeMB,
				IsCreated: true,
			},
		},
	}
}

func ResizeVolumePatch(volumeID string, sizeMB int64) EnvironmentConfig {
	return EnvironmentConfig{
		Volumes: map[string]VolumeConfig{
			volumeID: {
				SizeMB: sizeMB,
			},
		},
	}
}

func (c EnvironmentConfig) Bucket(bucketID string) (BucketConfig, bool) {
	bucket, ok := c.Buckets[bucketID]
	return bucket, ok
}

func DeleteServiceInstancePatch(serviceID string) EnvironmentConfig {
	return EnvironmentConfig{
		Services: map[string]ServiceConfig{
			serviceID: {
				IsDeleted: true,
			},
		},
	}
}

func variablePatch(serviceID string, name string, variable *VariableConfig) EnvironmentConfig {
	return EnvironmentConfig{
		Services: map[string]ServiceConfig{
			serviceID: {
				Variables: map[string]*VariableConfig{
					name: variable,
				},
			},
		},
	}
}
