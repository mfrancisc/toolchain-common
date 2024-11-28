package spaceprovisionerconfig

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/assertions"
	corev1 "k8s.io/api/core/v1"
)

type (
	readyWithStatusAndReason struct {
		expectedStatus corev1.ConditionStatus
		expectedReason *string
	}

	consumedSpaceCount struct {
		expectedSpaceCount int
	}

	consumedMemoryUsage struct {
		expectedMemoryUsage map[string]int
	}

	unknownConsumedCapacity struct{}
)

var (
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*readyWithStatusAndReason)(nil)
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*consumedSpaceCount)(nil)
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*consumedMemoryUsage)(nil)
	_ assertions.PredicateMatchFixer[*toolchainv1alpha1.SpaceProvisionerConfig] = (*unknownConsumedCapacity)(nil)
)

func (r *readyWithStatusAndReason) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	cond, found := condition.FindConditionByType(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
	if !found {
		return false
	}

	if cond.Status != r.expectedStatus {
		return false
	}

	if r.expectedReason != nil && cond.Reason != *r.expectedReason {
		return false
	}

	return true
}

func (r *readyWithStatusAndReason) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	cnd, found := condition.FindConditionByType(spc.Status.Conditions, toolchainv1alpha1.ConditionReady)
	cnd.Type = toolchainv1alpha1.ConditionReady
	cnd.Status = r.expectedStatus
	if r.expectedReason != nil {
		cnd.Reason = *r.expectedReason
	}
	if !found {
		spc.Status.Conditions = condition.AddStatusConditions(spc.Status.Conditions, cnd)
	} else {
		spc.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(spc.Status.Conditions, cnd)
	}
	return spc
}

func (p *consumedSpaceCount) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	if spc.Status.ConsumedCapacity == nil {
		return false
	}
	return p.expectedSpaceCount == spc.Status.ConsumedCapacity.SpaceCount
}

func (p *consumedSpaceCount) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	if spc.Status.ConsumedCapacity == nil {
		spc.Status.ConsumedCapacity = &toolchainv1alpha1.ConsumedCapacity{}
	}
	spc.Status.ConsumedCapacity.SpaceCount = p.expectedSpaceCount
	return spc
}

func (p *consumedMemoryUsage) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	if spc.Status.ConsumedCapacity == nil {
		return false
	}
	if len(spc.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole) != len(p.expectedMemoryUsage) {
		return false
	}
	for k, v := range spc.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole {
		if p.expectedMemoryUsage[k] != v {
			return false
		}
	}
	return true
}

func (p *consumedMemoryUsage) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	if spc.Status.ConsumedCapacity == nil {
		spc.Status.ConsumedCapacity = &toolchainv1alpha1.ConsumedCapacity{}
	}
	spc.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole = p.expectedMemoryUsage
	return spc
}

func (p *unknownConsumedCapacity) Matches(spc *toolchainv1alpha1.SpaceProvisionerConfig) bool {
	return spc.Status.ConsumedCapacity == nil
}

func (p *unknownConsumedCapacity) FixToMatch(spc *toolchainv1alpha1.SpaceProvisionerConfig) *toolchainv1alpha1.SpaceProvisionerConfig {
	spc.Status.ConsumedCapacity = nil
	return spc
}

func Ready() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &readyWithStatusAndReason{expectedStatus: corev1.ConditionTrue}
}

func NotReady() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &readyWithStatusAndReason{expectedStatus: corev1.ConditionFalse}
}

func NotReadyWithReason(reason string) assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &readyWithStatusAndReason{expectedStatus: corev1.ConditionFalse, expectedReason: &reason}
}

func ReadyStatusAndReason(status corev1.ConditionStatus, reason string) assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &readyWithStatusAndReason{expectedStatus: status, expectedReason: &reason}
}

func ConsumedSpaceCount(value int) assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &consumedSpaceCount{expectedSpaceCount: value}
}

func ConsumedMemoryUsage(values map[string]int) assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &consumedMemoryUsage{expectedMemoryUsage: values}
}

func UnknownConsumedCapacity() assertions.Predicate[*toolchainv1alpha1.SpaceProvisionerConfig] {
	return &unknownConsumedCapacity{}
}
