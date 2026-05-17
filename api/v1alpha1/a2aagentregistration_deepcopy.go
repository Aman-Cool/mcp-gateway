package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func (in *A2AAgentRegistration) DeepCopyInto(out *A2AAgentRegistration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *A2AAgentRegistration) DeepCopy() *A2AAgentRegistration {
	if in == nil {
		return nil
	}
	out := new(A2AAgentRegistration)
	in.DeepCopyInto(out)
	return out
}

func (in *A2AAgentRegistration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *A2AAgentRegistrationList) DeepCopyInto(out *A2AAgentRegistrationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]A2AAgentRegistration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *A2AAgentRegistrationList) DeepCopy() *A2AAgentRegistrationList {
	if in == nil {
		return nil
	}
	out := new(A2AAgentRegistrationList)
	in.DeepCopyInto(out)
	return out
}

func (in *A2AAgentRegistrationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *A2AAgentRegistrationSpec) DeepCopyInto(out *A2AAgentRegistrationSpec) {
	*out = *in
	out.TargetRef = in.TargetRef
	if in.CredentialRef != nil {
		in, out := &in.CredentialRef, &out.CredentialRef
		*out = new(SecretReference)
		**out = **in
	}
}

func (in *A2AAgentRegistrationSpec) DeepCopy() *A2AAgentRegistrationSpec {
	if in == nil {
		return nil
	}
	out := new(A2AAgentRegistrationSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *A2AAgentRegistrationStatus) DeepCopyInto(out *A2AAgentRegistrationStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *A2AAgentRegistrationStatus) DeepCopy() *A2AAgentRegistrationStatus {
	if in == nil {
		return nil
	}
	out := new(A2AAgentRegistrationStatus)
	in.DeepCopyInto(out)
	return out
}
