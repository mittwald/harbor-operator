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

	// Construct a new garbage collection from the provided instance spec
	newGc, err := BuildGarbageCollectionScheduleFromSpec(harbor)

	// Assume that the garbage collection's Schedule is nil, if it has not been created yet
	if gc.AdminJobSchedule == nil {
		err = harborClient.System().CreateSystemGarbageCollectionSchedule(newGc)
		if err != nil {
			return err
		}
		return nil
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

// BuildGarbageCollectionScheduleFromSpec constructs and returns a Harbor AdminJobReq from the spec
func BuildGarbageCollectionScheduleFromSpec(harbor *registriesv1alpha1.Instance) (h.AdminJobReq, error) {
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
	case h.ScheduleTypeCustom:
		return nil
	case h.ScheduleTypeHourly:
		return nil
	case h.ScheduleTypeDaily:
		return nil
	case h.ScheduleTypeWeekly:
		return nil
	case h.ScheduleTypeManual:
		return nil
	case h.ScheduleTypeNone:
		return nil
	}

	return errors.New(fmt.Sprintf("the provided garbage collection schedule type is invalid %s", param.Type))
}
