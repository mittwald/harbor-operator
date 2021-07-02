package internal

import (
	"strconv"

	"github.com/mittwald/goharbor-client/v4/apiv2/model"
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
)

// GenerateProjectMetadata constructs the project metadata for a Harbor project
func GenerateProjectMetadata(projectMeta *v1alpha2.ProjectMetadata) *model.ProjectMetadata {
	autoScan := strconv.FormatBool(projectMeta.AutoScan)
	enableContentTrust := strconv.FormatBool(projectMeta.EnableContentTrust)
	preventVul := strconv.FormatBool(projectMeta.PreventVul)
	reuseSysCVEAllowlist := strconv.FormatBool(projectMeta.ReuseSysCVEAllowlist)
	public := strconv.FormatBool(projectMeta.Public)

	pm := model.ProjectMetadata{
		AutoScan:             &autoScan,
		EnableContentTrust:   &enableContentTrust,
		PreventVul:           &preventVul,
		Public:               public,
		ReuseSysCveAllowlist: &reuseSysCVEAllowlist,
		Severity:             projectMeta.Severity,
	}

	return &pm
}
