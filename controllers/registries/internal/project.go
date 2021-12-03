package internal

import (
	"context"
	"errors"
	"strconv"

	h "github.com/mittwald/goharbor-client/v5/apiv2"
	"github.com/mittwald/goharbor-client/v5/apiv2/model"
	clienterrors "github.com/mittwald/goharbor-client/v5/apiv2/pkg/errors"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
)

func FetchHarborProjectIfExists(ctx context.Context, harborClient *h.RESTClient, projectName string) (*model.Project, bool, error) {
	p, err := harborClient.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(&clienterrors.ErrProjectUnknownResource{}, err) ||
			errors.Is(&clienterrors.ErrProjectNotFound{}, err) {
			return nil, false, nil
		}
		return p, false, err
	}

	return p, true, nil
}

func DeleteHarborProject(ctx context.Context, harborClient *h.RESTClient, p *model.Project) error {
	if err := harborClient.DeleteProject(ctx, p.Name); err != nil {
		if errors.Is(&clienterrors.ErrProjectMismatch{}, err) {
			return nil
		}
		if errors.Is(&clienterrors.ErrProjectNotFound{}, err) {
			return nil
		}
		return err
	}

	return nil
}

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
		ReuseSysCVEAllowlist: &reuseSysCVEAllowlist,
		Severity:             projectMeta.Severity,
	}

	return &pm
}
