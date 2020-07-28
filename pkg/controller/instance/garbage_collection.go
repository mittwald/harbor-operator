package instance

import (
	"context"
	"fmt"
	"reflect"

	system "github.com/mittwald/goharbor-client/system"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"

	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
)

// reconcileGarbageCollection reads the state of a configured garbage collection schedule and compares it to the user
// defined garbage collection schedule.
func (r *ReconcileInstance) reconcileGarbageCollection(ctx context.Context, harbor *registriesv1alpha1.Instance) error {
	scheduleType, err := enumGCType(harbor.Spec.GarbageCollection.ScheduleType)
	if err != nil {
		return err
	}

	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return err
	}

	newGc := modelv1.AdminJobSchedule{
		Schedule: &modelv1.AdminJobScheduleObj{
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
func enumGCType(receivedScheduleType registriesv1alpha1.ScheduleType) (registriesv1alpha1.ScheduleType, error) {
	switch receivedScheduleType {
	case registriesv1alpha1.ScheduleTypeCustom, registriesv1alpha1.ScheduleTypeDaily,
		registriesv1alpha1.ScheduleTypeHourly, registriesv1alpha1.ScheduleTypeManually,
		registriesv1alpha1.ScheduleTypeWeekly, registriesv1alpha1.ScheduleTypeNone:
		return receivedScheduleType, nil

	default:
		return "", fmt.Errorf("invalid garbage collection schedule type provided: '%s'", receivedScheduleType)
	}
}
