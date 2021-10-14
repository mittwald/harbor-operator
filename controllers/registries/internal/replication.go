package internal

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client/v4/apiv2"
	replicationapi "github.com/mittwald/goharbor-client/v4/apiv2/replication"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
)

// AssertDeletedReplication deletes a replication, first ensuring its existence
func AssertDeletedReplication(ctx context.Context, log logr.Logger,
	harborClient *h.RESTClient, replication *v1alpha2.Replication) error {
	receivedReplicationPolicy, err := harborClient.GetReplicationPolicy(ctx, replication.Name)
	if err != nil {
		if errors.Is(err, &replicationapi.ErrReplicationNotFound{}) {
			log.Info("replication does not exist on the server side, pulling finalizers")
			controllerutil.RemoveFinalizer(replication, FinalizerName)
			return nil
		}
		return err
	}

	err = harborClient.DeleteReplicationPolicy(ctx, receivedReplicationPolicy)
	if err != nil {
		if errors.Is(err, &replicationapi.ErrReplicationNotFound{}) {
			log.Info("replication does not exist on the server side, pulling finalizers")
			controllerutil.RemoveFinalizer(replication, FinalizerName)

		}
		return err
	}

	log.Info("replication was deleted, pulling finalizers")
	controllerutil.RemoveFinalizer(replication, FinalizerName)

	return nil
}
