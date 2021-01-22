package helper

import (
	"sort"

	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
)

func ToHarborRetentionPolicy(policy *registriesv1alpha2.RetentionPolicy) *legacymodel.RetentionPolicy {
	rp := &legacymodel.RetentionPolicy{
		Algorithm: policy.Algorithm,
		Scope: &legacymodel.RetentionPolicyScope{
			Level: policy.Scope.Level,
			Ref:   policy.Scope.Ref,
		},
		Trigger: &legacymodel.RetentionRuleTrigger{
			Kind:       policy.Trigger.Kind,
			References: policy.Trigger.References,
			Settings:   policy.Trigger.Settings,
		},
	}

	for _, r := range policy.Rules {
		var rpRule = legacymodel.RetentionRule{}
		rpRule.ID = r.ID
		rpRule.Template = r.Template
		rpRule.Action = r.Action
		rpRule.Disabled = r.Disabled
		rpRule.Params = r.Params
		rpRule.Priority = r.Priority
		rpRule.ScopeSelectors = ToHarborScopeSelectors(r.ScopeSelectors)
		rpRule.TagSelectors = ToHarborTagSelectors(r.TagSelectors)

		rp.Rules = append(rp.Rules, &rpRule)
	}

	return rp
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
