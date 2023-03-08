package registries

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/mittwald/goharbor-client/v5/apiv2/model"
	clienterrors "github.com/mittwald/goharbor-client/v5/apiv2/pkg/errors"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"
)

// reconcileGarbageCollection reads the state of a configured garbage collection schedule and compares it to the user
// defined garbage collection schedule.
func (r *InstanceReconciler) reconcileGarbageCollection(ctx context.Context, harbor *v1alpha2.Instance) error {
	scheduleType, err := enumGCType(harbor.Spec.GarbageCollection.ScheduleType)
	if err != nil {
		return err
	}

	harborClient, err := internal.BuildClient(ctx, r.Client, harbor)
	if err != nil {
		return err
	}

	var schedule string

	split := strings.Split(harbor.Spec.GarbageCollection.Cron, " ")
	if len(split) == 5 {
		schedule = strings.Join(append([]string{"*"}, split...), " ")
	} else {
		schedule = harbor.Spec.GarbageCollection.Cron
	}

	newGc := model.Schedule{
		Schedule: &model.ScheduleObj{
			Cron: schedule,
			Type: string(scheduleType),
		},
	}

	if harbor.Spec.GarbageCollection.DeleteUntagged {
		newGc.Parameters = map[string]interface{}{
			"delete_untagged": true,
		}
	}

	gc, err := harborClient.GetGarbageCollectionSchedule(ctx)
	if err != nil {
		if errors.Is(&clienterrors.ErrSystemGcScheduleUndefined{}, err) {
			// The initial GC schedule is always undefined, set it to the desired schedule.
			return harborClient.NewGarbageCollection(ctx, &newGc)
		}
		return err
	}

	// Compare the constructed garbage collection to the existing one and update accordingly
	if !reflect.DeepEqual(newGc.Schedule, gc.Schedule) {
		err = harborClient.UpdateGarbageCollection(ctx, &newGc)
		if err != nil {
			return err
		}
	}

	return nil
}

// enumGCType enumerates a string against valid GarbageCollection schedule types.
func enumGCType(receivedScheduleType v1alpha2.ScheduleType) (v1alpha2.ScheduleType, error) {
	switch receivedScheduleType {
	case v1alpha2.ScheduleTypeCustom, v1alpha2.ScheduleTypeDaily,
		v1alpha2.ScheduleTypeHourly, v1alpha2.ScheduleTypeManually,
		v1alpha2.ScheduleTypeWeekly, v1alpha2.ScheduleTypeNone:
		return receivedScheduleType, nil

	default:
		return "", fmt.Errorf("invalid garbage collection schedule type provided: '%s'", receivedScheduleType)
	}
}
