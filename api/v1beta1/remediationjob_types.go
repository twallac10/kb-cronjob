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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RemediationJobSpec defines the desired state of RemediationJob.
type RemediationJobSpec struct {
	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// Specifies the job that will be created when executing a CronJob.
	JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`

	// +kubebuilder:validation:Minimum=0

	// The number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// The number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`

	// This flag tells the controller what resources the remediation job should look for.

	Watch RemediationFilter `json:"watch"`
}

// ConcurrencyPolicy describes how the job will be handled.
// Only one of the following concurrent policies may be specified.
// If none of the following policies is specified, the default one
// is AllowConcurrent.
// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type ConcurrencyPolicy string

type RemediationFilter struct {
	// +kubebuilder:validation:Enum=Pod;Deployment;StatefulSet;DaemonSet;Job;CronJob;Service;ConfigMap;Secret;Ingress;PersistentVolumeClaim;PersistentVolume;StorageClass;Namespace;Role;RoleBinding;ClusterRole;ClusterRoleBinding;ServiceAccount;NetworkPolicy;PodSecurityPolicy;CustomResourceDefinition;MutatingWebhookConfiguration;ValidatingWebhookConfiguration;PodDisruptionBudget;HorizontalPodAutoscaler;PodSecurityPolicy;LimitRange;ResourceQuota;PriorityClass;PodPreset;PodTemplate;ReplicaSet;ReplicationController
	Kind string `json:"kind"`
	// +optional
	Prefix string `json:"prefix"`
	// +optional
	Suffix string `json:"suffix"`
	// +optional
	// kubebuilder should validate it's a list of strings that can only contain Create, Update, Delete
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:Enum=Create;Update;Delete
	Events []string `json:"events"`
	// +optional
	// +kubebuilder:validation:Minimum=0
	ResourceMinTrigger int32 `json:"resourceMinTrigger"`
}

const (
	// AllowConcurrent allows CronJobs to run concurrently.
	AllowConcurrent ConcurrencyPolicy = "Allow"

	// ForbidConcurrent forbids concurrent runs, skipping next run if previous
	// hasn't finished yet.
	ForbidConcurrent ConcurrencyPolicy = "Forbid"

	// ReplaceConcurrent cancels currently running job and replaces it with a new one.
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// RemediationJobStatus defines the observed state of RemediationJob.
type RemediationJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// A list of pointers to currently running jobs.
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RemediationJob is the Schema for the remediationjobs API.
type RemediationJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemediationJobSpec   `json:"spec,omitempty"`
	Status RemediationJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RemediationJobList contains a list of RemediationJob.
type RemediationJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemediationJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemediationJob{}, &RemediationJobList{})
}
