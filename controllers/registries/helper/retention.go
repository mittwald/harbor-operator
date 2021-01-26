package helper

import (
	"sort"

	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
)

// ToHarborRetentionPolicy constructs and returns a RetentionPolicy
// that can be passed to the Harbor API from a Retention CR's spec.
func ToHarborRetentionPolicy(policy *registriesv1alpha2.Retention, id *int64) *legacymodel.RetentionPolicy {
	rp := &legacymodel.RetentionPolicy{
		Algorithm: policy.Spec.Algorithm,
		Scope: &legacymodel.RetentionPolicyScope{
			Level: policy.Spec.Scope.Level,
			Ref:   policy.Spec.Scope.Ref,
		},
		Trigger: &legacymodel.RetentionRuleTrigger{
			Kind:       policy.Spec.Trigger.Kind,
			References: policy.Spec.Trigger.References,
			Settings:   policy.Spec.Trigger.Settings,
		},
	}

	if id != nil {
		rp.ID = *id
	}

	for _, r := range policy.Spec.Rules {
		rpRule := ToHarborRetentionPolicyRule(r)
		rp.Rules = append(rp.Rules, &rpRule)
	}

	return rp
}

func ToHarborRetentionPolicyRule(r registriesv1alpha2.RetentionRule) legacymodel.RetentionRule {
	var rpRule legacymodel.RetentionRule
	rpRule.ID = r.ID
	rpRule.Template = r.Template
	rpRule.Action = r.Action
	rpRule.Disabled = r.Disabled
	rpRule.Params = r.Params




	rpRule.Priority = r.Priority
	rpRule.ScopeSelectors = ToHarborScopeSelectors(r.ScopeSelectors)
	rpRule.TagSelectors = ToHarborTagSelectors(r.TagSelectors)
}

func ToHarborScopeSelectors(selectors map[string][]registriesv1alpha2.RetentionSelector) map[string][]legacymodel.RetentionSelector {
	var rs map[string][]legacymodel.RetentionSelector
	var sortedProps []string

	for s := range selectors {
		sortedProps = append(sortedProps, s)
	}
	sort.Strings(sortedProps)

	for _, p := range sortedProps {
		for key := range selectors[p] {
			rs[p][key] = legacymodel.RetentionSelector(selectors[p][key])
		}
	}

	return rs
}

func ToHarborTagSelectors(selectors []*registriesv1alpha2.RetentionSelector) []*legacymodel.RetentionSelector {
	var rs []*legacymodel.RetentionSelector

	for _, s := range selectors {
		rs = append(rs, &legacymodel.RetentionSelector{
			Decoration: s.Decoration,
			Extras:     s.Extras,
			Kind:       s.Kind,
			Pattern:    s.Pattern,
		})
	}

	return rs
}
