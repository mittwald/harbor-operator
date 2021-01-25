package registries

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mittwald/goharbor-client/v3/apiv2/system"
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"

	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
)

// reconcileGarbageCollection reads the state of a configured garbage collection schedule and compares it to the user
// defined garbage collection schedule.
func (r *InstanceReconciler) reconcileGarbageCollection(ctx context.Context, harbor *v1alpha2.Instance) error {
	scheduleType, err := enumGCType(harbor.Spec.GarbageCollection.ScheduleType)
	if err != nil {
		return err
	}

	harborClient, err := internal.BuildClient(ctx, r, harbor)
	if err != nil {
		return err
	}

	newGc := legacymodel.AdminJobSchedule{
		Schedule: &legacymodel.AdminJobScheduleObj{
			Cron: harbor.Spec.GarbageCollection.Cron,
			Type: string(scheduleType),
		},
	}

	gc, err := harborClient.GetSystemGarbageCollection(ctx)
	if err != nil && err.Error() == system.ErrSystemGcUndefinedMsg {
		if _, err := harborClient.NewSystemGarbageCollection(
			ctx,
			newGc.Schedule.Cron,
			newGc.Schedule.Type,
		); err != nil {
			return err
		}
	} else {
		return err
	}

	// Compare the constructed garbage collection to the existing one and update accordingly
	if !reflect.DeepEqual(newGc, gc) {
		err = harborClient.UpdateSystemGarbageCollection(ctx, newGc.Schedule)
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
