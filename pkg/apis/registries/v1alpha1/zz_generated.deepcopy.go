// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1alpha1

import (
	v1100 "github.com/mittwald/goharbor-client/model/v1_10_0"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GarbageCollection) DeepCopyInto(out *GarbageCollection) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GarbageCollection.
func (in *GarbageCollection) DeepCopy() *GarbageCollection {
	if in == nil {
		return nil
	}
	out := new(GarbageCollection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Instance) DeepCopyInto(out *Instance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Instance.
func (in *Instance) DeepCopy() *Instance {
	if in == nil {
		return nil
	}
	out := new(Instance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Instance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceChartRepo) DeepCopyInto(out *InstanceChartRepo) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceChartRepo.
func (in *InstanceChartRepo) DeepCopy() *InstanceChartRepo {
	if in == nil {
		return nil
	}
	out := new(InstanceChartRepo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InstanceChartRepo) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceChartRepoList) DeepCopyInto(out *InstanceChartRepoList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InstanceChartRepo, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceChartRepoList.
func (in *InstanceChartRepoList) DeepCopy() *InstanceChartRepoList {
	if in == nil {
		return nil
	}
	out := new(InstanceChartRepoList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InstanceChartRepoList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceChartRepoSpec) DeepCopyInto(out *InstanceChartRepoSpec) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceChartRepoSpec.
func (in *InstanceChartRepoSpec) DeepCopy() *InstanceChartRepoSpec {
	if in == nil {
		return nil
	}
	out := new(InstanceChartRepoSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceChartRepoStatus) DeepCopyInto(out *InstanceChartRepoStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceChartRepoStatus.
func (in *InstanceChartRepoStatus) DeepCopy() *InstanceChartRepoStatus {
	if in == nil {
		return nil
	}
	out := new(InstanceChartRepoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceDeployOptions) DeepCopyInto(out *InstanceDeployOptions) {
	*out = *in
	if in.KubeconfigSecret != nil {
		in, out := &in.KubeconfigSecret, &out.KubeconfigSecret
		*out = new(KubeconfigSecret)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceDeployOptions.
func (in *InstanceDeployOptions) DeepCopy() *InstanceDeployOptions {
	if in == nil {
		return nil
	}
	out := new(InstanceDeployOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceHelmChartSecretValues) DeepCopyInto(out *InstanceHelmChartSecretValues) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceHelmChartSecretValues.
func (in *InstanceHelmChartSecretValues) DeepCopy() *InstanceHelmChartSecretValues {
	if in == nil {
		return nil
	}
	out := new(InstanceHelmChartSecretValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceHelmChartSpec) DeepCopyInto(out *InstanceHelmChartSpec) {
	*out = *in
	out.ChartSpec = in.ChartSpec
	if in.SecretValues != nil {
		in, out := &in.SecretValues, &out.SecretValues
		*out = new(InstanceHelmChartSecretValues)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceHelmChartSpec.
func (in *InstanceHelmChartSpec) DeepCopy() *InstanceHelmChartSpec {
	if in == nil {
		return nil
	}
	out := new(InstanceHelmChartSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceList) DeepCopyInto(out *InstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Instance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceList.
func (in *InstanceList) DeepCopy() *InstanceList {
	if in == nil {
		return nil
	}
	out := new(InstanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InstanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceSpec) DeepCopyInto(out *InstanceSpec) {
	*out = *in
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = new(InstanceDeployOptions)
		(*in).DeepCopyInto(*out)
	}
	if in.HelmChart != nil {
		in, out := &in.HelmChart, &out.HelmChart
		*out = new(InstanceHelmChartSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.GarbageCollection != nil {
		in, out := &in.GarbageCollection, &out.GarbageCollection
		*out = new(GarbageCollection)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceSpec.
func (in *InstanceSpec) DeepCopy() *InstanceSpec {
	if in == nil {
		return nil
	}
	out := new(InstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceStatus) DeepCopyInto(out *InstanceStatus) {
	*out = *in
	in.Phase.DeepCopyInto(&out.Phase)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceStatus.
func (in *InstanceStatus) DeepCopy() *InstanceStatus {
	if in == nil {
		return nil
	}
	out := new(InstanceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceStatusPhase) DeepCopyInto(out *InstanceStatusPhase) {
	*out = *in
	if in.LastTransition != nil {
		in, out := &in.LastTransition, &out.LastTransition
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceStatusPhase.
func (in *InstanceStatusPhase) DeepCopy() *InstanceStatusPhase {
	if in == nil {
		return nil
	}
	out := new(InstanceStatusPhase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeconfigSecret) DeepCopyInto(out *KubeconfigSecret) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeconfigSecret.
func (in *KubeconfigSecret) DeepCopy() *KubeconfigSecret {
	if in == nil {
		return nil
	}
	out := new(KubeconfigSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MemberRequest) DeepCopyInto(out *MemberRequest) {
	*out = *in
	out.User = in.User
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MemberRequest.
func (in *MemberRequest) DeepCopy() *MemberRequest {
	if in == nil {
		return nil
	}
	out := new(MemberRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Registry) DeepCopyInto(out *Registry) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Registry.
func (in *Registry) DeepCopy() *Registry {
	if in == nil {
		return nil
	}
	out := new(Registry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Registry) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegistryList) DeepCopyInto(out *RegistryList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Registry, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegistryList.
func (in *RegistryList) DeepCopy() *RegistryList {
	if in == nil {
		return nil
	}
	out := new(RegistryList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RegistryList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegistrySpec) DeepCopyInto(out *RegistrySpec) {
	*out = *in
	if in.Credential != nil {
		in, out := &in.Credential, &out.Credential
		*out = new(v1100.RegistryCredential)
		**out = **in
	}
	out.ParentInstance = in.ParentInstance
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegistrySpec.
func (in *RegistrySpec) DeepCopy() *RegistrySpec {
	if in == nil {
		return nil
	}
	out := new(RegistrySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RegistryStatus) DeepCopyInto(out *RegistryStatus) {
	*out = *in
	if in.LastTransition != nil {
		in, out := &in.LastTransition, &out.LastTransition
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RegistryStatus.
func (in *RegistryStatus) DeepCopy() *RegistryStatus {
	if in == nil {
		return nil
	}
	out := new(RegistryStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Replication) DeepCopyInto(out *Replication) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Replication.
func (in *Replication) DeepCopy() *Replication {
	if in == nil {
		return nil
	}
	out := new(Replication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Replication) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicationFilter) DeepCopyInto(out *ReplicationFilter) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicationFilter.
func (in *ReplicationFilter) DeepCopy() *ReplicationFilter {
	if in == nil {
		return nil
	}
	out := new(ReplicationFilter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicationList) DeepCopyInto(out *ReplicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Replication, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicationList.
func (in *ReplicationList) DeepCopy() *ReplicationList {
	if in == nil {
		return nil
	}
	out := new(ReplicationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicationSpec) DeepCopyInto(out *ReplicationSpec) {
	*out = *in
	if in.SrcRegistry != nil {
		in, out := &in.SrcRegistry, &out.SrcRegistry
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.DestRegistry != nil {
		in, out := &in.DestRegistry, &out.DestRegistry
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Trigger != nil {
		in, out := &in.Trigger, &out.Trigger
		*out = new(ReplicationTrigger)
		(*in).DeepCopyInto(*out)
	}
	if in.Filters != nil {
		in, out := &in.Filters, &out.Filters
		*out = make([]ReplicationFilter, len(*in))
		copy(*out, *in)
	}
	out.ParentInstance = in.ParentInstance
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicationSpec.
func (in *ReplicationSpec) DeepCopy() *ReplicationSpec {
	if in == nil {
		return nil
	}
	out := new(ReplicationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicationStatus) DeepCopyInto(out *ReplicationStatus) {
	*out = *in
	if in.LastTransition != nil {
		in, out := &in.LastTransition, &out.LastTransition
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicationStatus.
func (in *ReplicationStatus) DeepCopy() *ReplicationStatus {
	if in == nil {
		return nil
	}
	out := new(ReplicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicationTrigger) DeepCopyInto(out *ReplicationTrigger) {
	*out = *in
	if in.Settings != nil {
		in, out := &in.Settings, &out.Settings
		*out = new(TriggerSettings)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicationTrigger.
func (in *ReplicationTrigger) DeepCopy() *ReplicationTrigger {
	if in == nil {
		return nil
	}
	out := new(ReplicationTrigger)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Repository) DeepCopyInto(out *Repository) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Repository.
func (in *Repository) DeepCopy() *Repository {
	if in == nil {
		return nil
	}
	out := new(Repository)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Repository) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryList) DeepCopyInto(out *RepositoryList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Repository, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryList.
func (in *RepositoryList) DeepCopy() *RepositoryList {
	if in == nil {
		return nil
	}
	out := new(RepositoryList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RepositoryList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryMetadata) DeepCopyInto(out *RepositoryMetadata) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryMetadata.
func (in *RepositoryMetadata) DeepCopy() *RepositoryMetadata {
	if in == nil {
		return nil
	}
	out := new(RepositoryMetadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositorySpec) DeepCopyInto(out *RepositorySpec) {
	*out = *in
	out.ParentInstance = in.ParentInstance
	out.Metadata = in.Metadata
	if in.MemberRequests != nil {
		in, out := &in.MemberRequests, &out.MemberRequests
		*out = make([]MemberRequest, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositorySpec.
func (in *RepositorySpec) DeepCopy() *RepositorySpec {
	if in == nil {
		return nil
	}
	out := new(RepositorySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryStatus) DeepCopyInto(out *RepositoryStatus) {
	*out = *in
	if in.LastTransition != nil {
		in, out := &in.LastTransition, &out.LastTransition
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryStatus.
func (in *RepositoryStatus) DeepCopy() *RepositoryStatus {
	if in == nil {
		return nil
	}
	out := new(RepositoryStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TriggerSettings) DeepCopyInto(out *TriggerSettings) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TriggerSettings.
func (in *TriggerSettings) DeepCopy() *TriggerSettings {
	if in == nil {
		return nil
	}
	out := new(TriggerSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *User) DeepCopyInto(out *User) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new User.
func (in *User) DeepCopy() *User {
	if in == nil {
		return nil
	}
	out := new(User)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *User) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserList) DeepCopyInto(out *UserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]User, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserList.
func (in *UserList) DeepCopy() *UserList {
	if in == nil {
		return nil
	}
	out := new(UserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *UserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserSpec) DeepCopyInto(out *UserSpec) {
	*out = *in
	out.ParentInstance = in.ParentInstance
	out.UserSecretRef = in.UserSecretRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserSpec.
func (in *UserSpec) DeepCopy() *UserSpec {
	if in == nil {
		return nil
	}
	out := new(UserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserStatus) DeepCopyInto(out *UserStatus) {
	*out = *in
	if in.LastTransition != nil {
		in, out := &in.LastTransition, &out.LastTransition
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserStatus.
func (in *UserStatus) DeepCopy() *UserStatus {
	if in == nil {
		return nil
	}
	out := new(UserStatus)
	in.DeepCopyInto(out)
	return out
}
