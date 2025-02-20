/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	autoremediationv1beta1 "t-wallace.com/kb-cronjob/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var remediationjoblog = logf.Log.WithName("remediationjob-resource")

// SetupRemediationJobWebhookWithManager registers the webhook for RemediationJob in the manager.
func SetupRemediationJobWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&autoremediationv1beta1.RemediationJob{}).
		WithValidator(&RemediationJobCustomValidator{}).
		WithDefaulter(&RemediationJobCustomDefaulter{
			DefaultConcurrencyPolicy:          autoremediationv1beta1.AllowConcurrent,
			DefaultSuspend:                    false,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
		}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-auto-remediation-t-wallace-com-v1beta1-remediationjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=auto-remediation.t-wallace.com,resources=remediationjobs,verbs=create;update,versions=v1beta1,name=mremediationjob-v1beta1.kb.io,admissionReviewVersions=v1

// RemediationJobCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind RemediationJob when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type RemediationJobCustomDefaulter struct {
	// Default values for various CronJob fields
	DefaultConcurrencyPolicy          autoremediationv1beta1.ConcurrencyPolicy
	DefaultSuspend                    bool
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit     int32
}

var _ webhook.CustomDefaulter = &RemediationJobCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind RemediationJob.
func (d *RemediationJobCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	remediationjob, ok := obj.(*autoremediationv1beta1.RemediationJob)

	if !ok {
		return fmt.Errorf("expected an RemediationJob object but got %T", obj)
	}
	remediationjoblog.Info("Defaulting for RemediationJob", "name", remediationjob.GetName())

	d.applyDefaults(remediationjob)

	return nil
}

func (d *RemediationJobCustomDefaulter) applyDefaults(remediationJob *autoremediationv1beta1.RemediationJob) {
	if remediationJob.Spec.ConcurrencyPolicy == "" {
		remediationJob.Spec.ConcurrencyPolicy = d.DefaultConcurrencyPolicy
	}
	if remediationJob.Spec.Suspend == nil {
		remediationJob.Spec.Suspend = new(bool)
		*remediationJob.Spec.Suspend = d.DefaultSuspend
	}
	if remediationJob.Spec.SuccessfulJobsHistoryLimit == nil {
		remediationJob.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*remediationJob.Spec.SuccessfulJobsHistoryLimit = d.DefaultSuccessfulJobsHistoryLimit
	}
	if remediationJob.Spec.FailedJobsHistoryLimit == nil {
		remediationJob.Spec.FailedJobsHistoryLimit = new(int32)
		*remediationJob.Spec.FailedJobsHistoryLimit = d.DefaultFailedJobsHistoryLimit
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-auto-remediation-t-wallace-com-v1beta1-remediationjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=auto-remediation.t-wallace.com,resources=remediationjobs,verbs=create;update,versions=v1beta1,name=vremediationjob-v1beta1.kb.io,admissionReviewVersions=v1

// RemediationJobCustomValidator struct is responsible for validating the RemediationJob resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type RemediationJobCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &RemediationJobCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type RemediationJob.
func (v *RemediationJobCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	remediationjob, ok := obj.(*autoremediationv1beta1.RemediationJob)
	if !ok {
		return nil, fmt.Errorf("expected a RemediationJob object but got %T", obj)
	}
	remediationjoblog.Info("Validation for RemediationJob upon creation", "name", remediationjob.GetName())

	return nil, validateRemediationJob(remediationjob)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type RemediationJob.
func (v *RemediationJobCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	remediationjob, ok := newObj.(*autoremediationv1beta1.RemediationJob)
	if !ok {
		return nil, fmt.Errorf("expected a RemediationJob object for the newObj but got %T", newObj)
	}
	remediationjoblog.Info("Validation for RemediationJob upon update", "name", remediationjob.GetName())

	return nil, validateRemediationJob(remediationjob)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type RemediationJob.
func (v *RemediationJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	remediationjob, ok := obj.(*autoremediationv1beta1.RemediationJob)
	if !ok {
		return nil, fmt.Errorf("expected a RemediationJob object but got %T", obj)
	}
	remediationjoblog.Info("Validation for RemediationJob upon deletion", "name", remediationjob.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

// validateRemediationJob validates the fields of a RemediationJob object.
func validateRemediationJob(Remediationjob *autoremediationv1beta1.RemediationJob) error {
	var allErrs field.ErrorList
	if err := validateRemediationJobName(Remediationjob); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateRemediationJobSpec(Remediationjob); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "RemediationJob"},
		Remediationjob.Name, allErrs)
}

func validateRemediationJobSpec(cronjob *autoremediationv1beta1.RemediationJob) *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	return nil
}

func validateRemediationJobName(remediationjob *autoremediationv1beta1.RemediationJob) *field.Error {
	if len(remediationjob.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// The job name length is 63 characters like all Kubernetes objects
		// (which must fit in a DNS subdomain). The cronjob controller appends
		// a 11-character suffix to the cronjob (`-$TIMESTAMP`) when creating
		// a job. The job name length limit is 63 characters. Therefore cronjob
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		return field.Invalid(field.NewPath("metadata").Child("name"), remediationjob.ObjectMeta.Name, "must be no more than 52 characters")
	}
	return nil
}
