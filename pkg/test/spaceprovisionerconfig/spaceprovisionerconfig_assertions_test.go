package spaceprovisionerconfig

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestReadyWithStatusAndReasonPredicate(t *testing.T) {
	test := func(status corev1.ConditionStatus, reason string, t *testing.T) {
		t.Run("matching: status="+string(status)+", reason='"+reason+"'", func(t *testing.T) {
			// given
			pred := &readyWithStatusAndReason{expectedStatus: status}
			spc := NewSpaceProvisionerConfig("spc", "default", WithReadyCondition(status, reason))

			// when & then
			assert.True(t, pred.Matches(spc))
		})

		t.Run("fixer with no conditions: status="+string(status)+", reason='"+reason+"'", func(t *testing.T) {
			// given
			pred := &readyWithStatusAndReason{expectedStatus: status}
			spc := NewSpaceProvisionerConfig("spc", "default")

			// when
			spc = pred.FixToMatch(spc)

			// then
			assert.Equal(t, 1, condition.Count(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, status, ""))
		})
		t.Run("fixer with different conditions: status="+string(status)+", reason='"+reason+"'", func(t *testing.T) {
			// given
			pred := &readyWithStatusAndReason{expectedStatus: status}
			spc := NewSpaceProvisionerConfig("spc", "default")
			spc.Status.Conditions = []toolchainv1alpha1.Condition{
				{
					Type:   toolchainv1alpha1.ConditionType("made up"),
					Status: corev1.ConditionTrue,
				},
			}

			// when
			spc = pred.FixToMatch(spc)

			// then
			assert.Equal(t, 1, condition.Count(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, status, ""))
			assert.Len(t, spc.Status.Conditions, 2)
		})
		t.Run("fixer with wrong condition: status="+string(status)+", reason='"+reason+"'", func(t *testing.T) {
			// given
			anotherStatus := corev1.ConditionTrue
			if anotherStatus == status {
				anotherStatus = corev1.ConditionFalse
			}
			if anotherStatus == status {
				anotherStatus = corev1.ConditionUnknown
			}
			pred := &readyWithStatusAndReason{expectedStatus: status}
			expectedReason := "because"
			spc := NewSpaceProvisionerConfig("spc", "default", WithReadyCondition(anotherStatus, expectedReason))

			// when
			spc = pred.FixToMatch(spc)

			// then
			assert.Equal(t, 1, condition.Count(spc.Status.Conditions, toolchainv1alpha1.ConditionReady, status, expectedReason))
		})
	}

	for _, status := range []corev1.ConditionStatus{corev1.ConditionTrue, corev1.ConditionFalse, corev1.ConditionUnknown} {
		t.Run("no reason specified", func(t *testing.T) {
			test(status, "", t)
		})
		t.Run("with reason", func(t *testing.T) {
			test(status, "the reason", t)
		})
	}
}

func TestConsumedSpaceCountPredicate(t *testing.T) {
	t.Run("matches", func(t *testing.T) {
		// given
		pred := &consumedSpaceCount{expectedSpaceCount: 5}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedSpaceCount(5))

		// when & then
		assert.True(t, pred.Matches(spc))
	})
	t.Run("doesn't match", func(t *testing.T) {
		// given
		pred := &consumedSpaceCount{expectedSpaceCount: 5}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedSpaceCount(4))

		// when & then
		assert.False(t, pred.Matches(spc))
	})
	t.Run("doesn't match if nil consumed capacity", func(t *testing.T) {
		// given
		pred := &consumedSpaceCount{expectedSpaceCount: 5}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when & then
		assert.False(t, pred.Matches(spc))
	})
	t.Run("fixes wrong value", func(t *testing.T) {
		// given
		pred := &consumedSpaceCount{expectedSpaceCount: 5}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedSpaceCount(4))

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.NotNil(t, fixed.Status.ConsumedCapacity)
		assert.Equal(t, 5, fixed.Status.ConsumedCapacity.SpaceCount)
	})
	t.Run("fixes if there's no consumed capacity", func(t *testing.T) {
		// given
		pred := &consumedSpaceCount{expectedSpaceCount: 5}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.NotNil(t, fixed.Status.ConsumedCapacity)
		assert.Equal(t, 5, fixed.Status.ConsumedCapacity.SpaceCount)
	})
}

func TestConsumedMemoryUsagePredicate(t *testing.T) {
	t.Run("matches", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedMemoryUsagePercentInNode("worker", 5), WithConsumedMemoryUsagePercentInNode("master", 20))

		// when & then
		assert.True(t, pred.Matches(spc))
	})
	t.Run("doesn't match if expecting more keys", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedMemoryUsagePercentInNode("worker", 5))

		// when & then
		assert.False(t, pred.Matches(spc))
	})
	t.Run("doesn't match if more keys present", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5}}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedMemoryUsagePercentInNode("worker", 5), WithConsumedMemoryUsagePercentInNode("master", 20))

		// when & then
		assert.False(t, pred.Matches(spc))
	})
	t.Run("doesn't match if value differs", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedMemoryUsagePercentInNode("worker", 5), WithConsumedMemoryUsagePercentInNode("master", 21))

		// when & then
		assert.False(t, pred.Matches(spc))
	})
	t.Run("doesn't match if nil consumed capacity", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when & then
		assert.False(t, pred.Matches(spc))
	})
	t.Run("fixes no consumed capacity", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.NotNil(t, fixed.Status.ConsumedCapacity)
		assert.Len(t, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole, 2)
		assert.Equal(t, 5, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole["worker"])
		assert.Equal(t, 20, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole["master"])
	})
	t.Run("fixes different values", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default",
			WithConsumedMemoryUsagePercentInNode("worker", 6),
			WithConsumedMemoryUsagePercentInNode("master", 21),
		)

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.NotNil(t, fixed.Status.ConsumedCapacity)
		assert.Len(t, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole, 2)
		assert.Equal(t, 5, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole["worker"])
		assert.Equal(t, 20, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole["master"])
	})
	t.Run("fixes different keys", func(t *testing.T) {
		// given
		pred := &consumedMemoryUsage{expectedMemoryUsage: map[string]int{"worker": 5, "master": 20}}
		spc := NewSpaceProvisionerConfig("spc", "default",
			WithConsumedMemoryUsagePercentInNode("master", 21),
			WithConsumedMemoryUsagePercentInNode("disaster", 80),
		)

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.NotNil(t, fixed.Status.ConsumedCapacity)
		assert.Len(t, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole, 2)
		assert.Equal(t, 5, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole["worker"])
		assert.Equal(t, 20, fixed.Status.ConsumedCapacity.MemoryUsagePercentPerNodeRole["master"])
	})
}

func TestUnknownConsumedCapacityPredicate(t *testing.T) {
	t.Run("matches", func(t *testing.T) {
		pred := &unknownConsumedCapacity{}
		spc := NewSpaceProvisionerConfig("spc", "default")

		assert.True(t, pred.Matches(spc))
	})
	t.Run("doesn't match with space count", func(t *testing.T) {
		pred := &unknownConsumedCapacity{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedSpaceCount(5))

		assert.False(t, pred.Matches(spc))
	})
	t.Run("doesn't match with memory usage", func(t *testing.T) {
		pred := &unknownConsumedCapacity{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedMemoryUsagePercentInNode("master", 5))

		assert.False(t, pred.Matches(spc))
	})
	t.Run("doesn't match with full consumed capacity", func(t *testing.T) {
		pred := &unknownConsumedCapacity{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedSpaceCount(5), WithConsumedMemoryUsagePercentInNode("workter", 5))

		assert.False(t, pred.Matches(spc))
	})
	t.Run("fixer does nothing on matching SPC", func(t *testing.T) {
		// given
		pred := &unknownConsumedCapacity{}
		spc := NewSpaceProvisionerConfig("spc", "default")

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.Equal(t, spc, fixed)
	})
	t.Run("fixes non-matching", func(t *testing.T) {
		// given
		pred := &unknownConsumedCapacity{}
		spc := NewSpaceProvisionerConfig("spc", "default", WithConsumedSpaceCount(5))

		// when
		fixed := pred.FixToMatch(spc.DeepCopy())

		// then
		assert.NotEqual(t, spc, fixed)
		assert.Nil(t, fixed.Status.ConsumedCapacity)
	})
}
