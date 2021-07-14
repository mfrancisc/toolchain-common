package config

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
)

type EnvName string

const (
	Prod EnvName = "prod"
	E2E  EnvName = "e2e-tests"
	Dev  EnvName = "dev"
)

type ToolchainConfigOptionFunc func(config *toolchainv1alpha1.ToolchainConfig)

type ToolchainConfigOption interface {
	Apply(config *toolchainv1alpha1.ToolchainConfig)
}

type ToolchainConfigOptionImpl struct {
	toApply []ToolchainConfigOptionFunc
}

func (option *ToolchainConfigOptionImpl) Apply(config *toolchainv1alpha1.ToolchainConfig) {
	for _, apply := range option.toApply {
		apply(config)
	}
}

func (option *ToolchainConfigOptionImpl) addFunction(funcToAdd ToolchainConfigOptionFunc) {
	option.toApply = append(option.toApply, funcToAdd)
}

type PerMemberClusterOptionInt func(map[string]int)

func PerMemberCluster(name string, value int) PerMemberClusterOptionInt {
	return func(clusters map[string]int) {
		clusters[name] = value
	}
}

//---Host Configurations---//

type EnvironmentOption struct {
	*ToolchainConfigOptionImpl
}

// Environments: Prod, E2E, Dev
func Environment(value EnvName) *EnvironmentOption {
	o := &EnvironmentOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		val := string(value)
		config.Spec.Host.Environment = &val
	})
	return o
}

type AutomaticApprovalOption struct {
	*ToolchainConfigOptionImpl
}

func AutomaticApproval() *AutomaticApprovalOption {
	o := &AutomaticApprovalOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval = toolchainv1alpha1.AutomaticApprovalConfig{}
	})
	return o
}

func (o AutomaticApprovalOption) Enabled(value bool) AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval.Enabled = &value
	})
	return o
}

func (o AutomaticApprovalOption) ResourceCapacityThreshold(defaultThreshold int, perMember ...PerMemberClusterOptionInt) AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.DefaultThreshold = &defaultThreshold
		config.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster = map[string]int{}
		for _, add := range perMember {
			add(config.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster)
		}
	})
	return o
}

func (o AutomaticApprovalOption) MaxNumberOfUsers(overall int, perMember ...PerMemberClusterOptionInt) AutomaticApprovalOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.AutomaticApproval.MaxNumberOfUsers.Overall = &overall
		config.Spec.Host.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster = map[string]int{}
		for _, add := range perMember {
			add(config.Spec.Host.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster)
		}
	})
	return o
}

type DeactivationOption struct {
	*ToolchainConfigOptionImpl
}

func Deactivation() *DeactivationOption {
	o := &DeactivationOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation = toolchainv1alpha1.DeactivationConfig{}
	})
	return o
}

func (o DeactivationOption) DeactivatingNotificationDays(value int) DeactivationOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation.DeactivatingNotificationDays = &value
	})
	return o
}

func (o DeactivationOption) DeactivationDomainsExcluded(value string) DeactivationOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation.DeactivationDomainsExcluded = &value
	})
	return o
}

func (o DeactivationOption) UserSignupDeactivatedRetentionDays(value int) DeactivationOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation.UserSignupDeactivatedRetentionDays = &value
	})
	return o
}

func (o DeactivationOption) UserSignupUnverifiedRetentionDays(value int) DeactivationOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Deactivation.UserSignupUnverifiedRetentionDays = &value
	})
	return o
}

type MetricsOption struct {
	*ToolchainConfigOptionImpl
}

func Metrics() *MetricsOption {
	o := &MetricsOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Metrics = toolchainv1alpha1.MetricsConfig{}
	})
	return o
}

func (o MetricsOption) ForceSynchronization(value bool) MetricsOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Metrics.ForceSynchronization = &value
	})
	return o
}

type NotificationsOption struct {
	*ToolchainConfigOptionImpl
}

func Notifications() *NotificationsOption {
	o := &NotificationsOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications = toolchainv1alpha1.NotificationsConfig{}
	})
	return o
}

func (o NotificationsOption) NotificationDeliveryService(value string) NotificationsOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.NotificationDeliveryService = &value
	})
	return o
}

func (o NotificationsOption) DurationBeforeNotificationDeletion(value string) NotificationsOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.DurationBeforeNotificationDeletion = &value
	})
	return o
}

func (o NotificationsOption) AdminEmail(value string) NotificationsOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.AdminEmail = &value
	})
	return o
}

type NotificationSecretOption struct {
	*ToolchainConfigOptionImpl
}

func (o NotificationsOption) Secret() *NotificationSecretOption {
	c := &NotificationSecretOption{
		ToolchainConfigOptionImpl: o.ToolchainConfigOptionImpl,
	}
	return c
}

func (o NotificationSecretOption) Ref(value string) NotificationSecretOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.Secret.Ref = &value
	})
	return o
}

func (o NotificationSecretOption) MailgunDomain(value string) NotificationSecretOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.Secret.MailgunDomain = &value
	})
	return o
}

func (o NotificationSecretOption) MailgunAPIKey(value string) NotificationSecretOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.Secret.MailgunAPIKey = &value
	})
	return o
}

func (o NotificationSecretOption) MailgunSenderEmail(value string) NotificationSecretOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.Secret.MailgunSenderEmail = &value
	})
	return o
}

func (o NotificationSecretOption) MailgunReplyToEmail(value string) NotificationSecretOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Notifications.Secret.MailgunReplyToEmail = &value
	})
	return o
}

type RegistrationServiceOption struct {
	*ToolchainConfigOptionImpl
}

func RegistrationService() *RegistrationServiceOption {
	o := &RegistrationServiceOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.RegistrationService = toolchainv1alpha1.RegistrationServiceConfig{}
	})
	return o
}

func (o RegistrationServiceOption) RegistrationServiceURL(value string) RegistrationServiceOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.RegistrationService.RegistrationServiceURL = &value
	})
	return o
}

type TiersOption struct {
	*ToolchainConfigOptionImpl
}

func Tiers() *TiersOption {
	o := &TiersOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Tiers = toolchainv1alpha1.TiersConfig{}
	})
	return o
}

func (o TiersOption) DurationBeforeChangeTierRequestDeletion(value string) TiersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Tiers.DurationBeforeChangeTierRequestDeletion = &value
	})
	return o
}

func (o TiersOption) TemplateUpdateRequestMaxPoolSize(value int) TiersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Tiers.TemplateUpdateRequestMaxPoolSize = &value
	})
	return o
}

type ToolchainStatusOption struct {
	*ToolchainConfigOptionImpl
}

func ToolchainStatus() *ToolchainStatusOption {
	o := &ToolchainStatusOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.ToolchainStatus = toolchainv1alpha1.ToolchainStatusConfig{}
	})
	return o
}

func (o ToolchainStatusOption) ToolchainStatusRefreshTime(value string) ToolchainStatusOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.ToolchainStatus.ToolchainStatusRefreshTime = &value
	})
	return o
}

type UsersOption struct {
	*ToolchainConfigOptionImpl
}

func Users() *UsersOption {
	o := &UsersOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Users = toolchainv1alpha1.UsersConfig{}
	})
	return o
}

func (o UsersOption) MasterUserRecordUpdateFailureThreshold(value int) UsersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Users.MasterUserRecordUpdateFailureThreshold = &value
	})
	return o
}

func (o UsersOption) ForbiddenUsernamePrefixes(value string) UsersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Users.ForbiddenUsernamePrefixes = &value
	})
	return o
}

func (o UsersOption) ForbiddenUsernameSuffixes(value string) UsersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Host.Users.ForbiddenUsernameSuffixes = &value
	})
	return o
}

//---End of Host Configurations---//

//---Member Configurations---//
type MembersOption struct {
	*ToolchainConfigOptionImpl
}

func Members() *MembersOption {
	o := &MembersOption{
		ToolchainConfigOptionImpl: &ToolchainConfigOptionImpl{},
	}
	return o
}

func (o MembersOption) Default(memberConfigSpec toolchainv1alpha1.MemberOperatorConfigSpec) MembersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		config.Spec.Members.Default = memberConfigSpec
	})
	return o
}

func (o MembersOption) SpecificPerMemberCluster(clusterName string, memberConfigSpec toolchainv1alpha1.MemberOperatorConfigSpec) MembersOption {
	o.addFunction(func(config *toolchainv1alpha1.ToolchainConfig) {
		if config.Spec.Members.SpecificPerMemberCluster == nil {
			config.Spec.Members.SpecificPerMemberCluster = make(map[string]toolchainv1alpha1.MemberOperatorConfigSpec)
		}
		config.Spec.Members.SpecificPerMemberCluster[clusterName] = memberConfigSpec
	})
	return o
}

//---End of Member Configurations---//
