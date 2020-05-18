package instance

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
)

func (r *ReconcileInstance) reconcileGarbageCollection(ctx context.Context, harbor *registriesv1alpha1.Instance) error {
	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return err
	}

	// Get the held garbage collection (Harbor AdminJob) from the Harbor API
	// While the API returns HTTP status 200 if garbage collection is not yet configured,
	// the object received is an empty h.AdminJobReq{}
	gc, err := harborClient.System().GetSystemGarbageCollectionSchedule()
	if err != nil {
		return err
	}

	// Construct a new garbage collection from the provided instance spec
	newGc, err := buildGarbageCollectionScheduleFromSpec(harbor)
	if err != nil {
		return err
	}

	// Assume that the garbage collection's Schedule is nil, if it has not been created yet
	if gc.AdminJobSchedule == nil {
		return harborClient.System().CreateSystemGarbageCollectionSchedule(newGc)
	}

	// Compare the constructed garbage collection to the existing one and update accordingly
	if !reflect.DeepEqual(newGc, gc) {
		err = harborClient.System().UpdateSystemGarbageCollectionSchedule(newGc)
		if err != nil {
			return err
		}
	}

	return nil
}

// buildGarbageCollectionScheduleFromSpec constructs and returns a Harbor AdminJobReq from the spec
func buildGarbageCollectionScheduleFromSpec(harbor *registriesv1alpha1.Instance) (h.AdminJobReq, error) {
	if harbor.Spec.GarbageCollection.Schedule == nil {
		return h.AdminJobReq{}, errors.New("no garbage collection schedule provided")
	}

	err := enumScheduleParam(harbor.Spec.GarbageCollection.Schedule)
	if err != nil {
		return h.AdminJobReq{}, err
	}

	gc := h.AdminJobReq{
		AdminJobSchedule: &h.ScheduleParam{
			Type: harbor.Spec.GarbageCollection.Schedule.Type,
			Cron: harbor.Spec.GarbageCollection.Schedule.Cron,
		},
	}

	if harbor.Spec.GarbageCollection.Parameters != nil {
		m := make(map[string]interface{})
		for k, v := range harbor.Spec.GarbageCollection.Parameters {
			m[k] = v
		}
		gc.Parameters = m
	}

	return gc, nil
}

// enumScheduleParam enumerates the provided schedule parameters for a garbage collection schedule (Harbor AdminJob)
func enumScheduleParam(param *h.ScheduleParam) error {
	if param.Cron == "" {
		return errors.New("the provided garbage collection cron schedule is empty")
	}

	switch param.Type {
	case h.ScheduleTypeCustom, h.ScheduleTypeHourly, h.ScheduleTypeDaily, h.ScheduleTypeWeekly, h.ScheduleTypeManual, h.ScheduleTypeNone:
		return nil
	default:
		return fmt.Errorf("the provided garbage collection schedule type is invalid %s", param.Type)
	}
}
