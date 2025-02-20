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

package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	autoremediationv1beta1 "t-wallace.com/kb-cronjob/api/v1beta1"
)

// RemediationJobReconciler reconciles a RemediationJob object
type RemediationJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=auto-remediation.t-wallace.com,resources=remediationjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auto-remediation.t-wallace.com,resources=remediationjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auto-remediation.t-wallace.com,resources=remediationjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RemediationJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *RemediationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var remediationJob autoremediationv1beta1.RemediationJob
	if err := r.Get(ctx, req.NamespacedName, &remediationJob); err != nil {
		log.Error(err, "unable to fetch RemediationJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var childJobs batch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	var activeJobs []*batch.Job
	var successfulJobs []*batch.Job
	var failedJobs []*batch.Job
	var mostRecentTime *time.Time // find the last run so we can update the status

	isJobFinished := func(job *batch.Job) (bool, batch.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == batch.JobComplete || c.Type == batch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case batch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case batch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}
	}

	if mostRecentTime != nil {
		remediationJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		remediationJob.Status.LastScheduleTime = nil
	}
	remediationJob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		remediationJob.Status.Active = append(remediationJob.Status.Active, *jobRef)
	}

	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	if err := r.Status().Update(ctx, &remediationJob); err != nil {
		log.Error(err, "unable to update remediationJob status")
		return ctrl.Result{}, err
	}

	// NB: deleting these are "best effort" -- if we fail on a particular one,
	// we won't requeue just to finish the deleting.
	if remediationJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*remediationJob.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			} else {
				log.V(0).Info("deleted old failed job", "job", job)
			}
		}
	}

	if remediationJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if int32(i) >= int32(len(successfulJobs))-*remediationJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			} else {
				log.V(0).Info("deleted old successful job", "job", job)
			}
		}
	}

	// Check if the job is suspended
	if remediationJob.Spec.Suspend != nil && *remediationJob.Spec.Suspend {
		log.V(1).Info("remediationJob is suspended, skipping")
		return ctrl.Result{}, nil
	}

	// Check if the job has active jobs
	if len(activeJobs) > 0 {
		log.V(1).Info("remediationJob has active jobs, skipping")
		return ctrl.Result{}, nil
	}

	// Check if the watch event is set for Delete
	for _, event := range remediationJob.Spec.Watch.Events {
		var count int32 = 0
		if event == "Delete" {
			switch remediationJob.Spec.Watch.Kind {
			case "Secret":
				secList := &corev1.SecretList{}
				if err := r.List(ctx, secList, client.InNamespace(req.Namespace)); err != nil {
					log.Error(err, "unable to list secrets")
					return ctrl.Result{}, err
				}
				for _, sec := range secList.Items {
					if remediationJob.Spec.Watch.Prefix != "" && strings.HasPrefix(sec.Name, remediationJob.Spec.Watch.Prefix) {
						log.Info("found secret")
						count++
					} else if remediationJob.Spec.Watch.Suffix != "" && strings.HasSuffix(sec.Name, remediationJob.Spec.Watch.Suffix) {
						log.Info("found secret")
						count++
					}
				}
				if count <= remediationJob.Spec.Watch.ResourceMinTrigger {
					log.Info("found secret limit reached, running job")
				} else {
					log.Info("found secret limit not reached, skipping")
					return ctrl.Result{}, nil
				}
			}
		}
	}

	// figure out how to run this job -- concurrency policy might forbid us from running
	// multiple at the same time...
	if remediationJob.Spec.ConcurrencyPolicy == autoremediationv1beta1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return ctrl.Result{}, nil
	}

	// ...or instruct us to replace existing ones...
	if remediationJob.Spec.ConcurrencyPolicy == autoremediationv1beta1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			// we don't care if the job was already deleted
			if err := r.Delete(ctx, activeJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}

	constructJobForRemediationJob := func(remediationJob *autoremediationv1beta1.RemediationJob) (*batch.Job, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		name := fmt.Sprintf("%s-%s", remediationJob.Name, rand.String(5))

		job := &batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   remediationJob.Namespace,
			},
			Spec: *remediationJob.Spec.JobTemplate.Spec.DeepCopy(),
		}
		for k, v := range remediationJob.Spec.JobTemplate.Annotations {
			job.Annotations[k] = v
		}
		// job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
		for k, v := range remediationJob.Spec.JobTemplate.Labels {
			job.Labels[k] = v
		}
		if err := ctrl.SetControllerReference(remediationJob, job, r.Scheme); err != nil {
			return nil, err
		}

		return job, nil
	}
	// actually make the job...
	job, err := constructJobForRemediationJob(&remediationJob)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}

	// ...and create it on the cluster
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for CronJob run", "job", job)

	return ctrl.Result{}, nil
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = autoremediationv1beta1.GroupVersion.String()
)

// // SetupWithManager sets up the controller with the Manager.
func (r *RemediationJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batch.Job{}, jobOwnerKey, func(rawObj client.Object) []string {
		job := rawObj.(*batch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "RemediationJob" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	updatePred := predicate.Funcs{
		// Allow delete events
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&autoremediationv1beta1.RemediationJob{}).
		//Watch for deletions of secrets and trigger a RemediationJob
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, cm client.Object) []ctrl.Request {
				// Get the namespace of the secret
				namespace := cm.GetNamespace()
				// Get the name of the secret
				secretName := cm.GetName()
				// Get the RemediationJob
				remediationJobList := &autoremediationv1beta1.RemediationJobList{}
				// List all RemediationJobs in the namespace
				if err := r.List(ctx, remediationJobList, client.InNamespace(namespace)); err != nil {
					return nil
				}
				// Create a list of requests to return
				var requests []ctrl.Request
				// Iterate over all RemediationJobs
				for _, remediationJob := range remediationJobList.Items {
					// Check if the RemediationJob has a secret with the same name
					if strings.HasPrefix(secretName, "gapi") {
						// Append the RemediationJob to the list of requests
						requests = append(requests, ctrl.Request{
							NamespacedName: types.NamespacedName{
								Namespace: remediationJob.Namespace,
								Name:      remediationJob.Name,
							},
						})
					}
				}
				// Return the list of requests
				return requests
			}),
			builder.WithPredicates(updatePred),
		).
		Complete(r)
}
