// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	logger "log"
	"strings"
	"sync"
	"time"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/pods"
	taskrunpkg "github.com/tektoncd/cli/pkg/taskrun"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	MsgTRNotFoundErr = "Unable to get TaskRun"
)

type step struct {
	name      string
	container string
	state     corev1.ContainerState
}

func (s *step) hasStarted() bool {
	return s.state.Waiting == nil
}

func (r *Reader) readTaskLog() (<-chan Log, <-chan error, error) {
	// 这个地方channel 没有关闭
	logger.Println("PipelineRun Log readTaskLog start")
	defer func() {
		logger.Println("PipelineRun Log readTaskLog end")
	}()
	tr, err := taskrunpkg.GetTaskRun(taskrunGroupResource, r.clients, r.run, r.ns)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %s", MsgTRNotFoundErr, err)
	}

	r.formTaskName(tr)
	// todo: 如果 pod 不存在, 这个时候获取日志的时候就会出错了.

	// todo: xxxx
	if !tr.IsDone() && r.follow {
		return r.readLiveTaskLogs(tr)
	}
	return r.readAvailableTaskLogs(tr)
}

func (r *Reader) formTaskName(tr *v1.TaskRun) {
	if r.task != "" {
		return
	}

	if name, ok := tr.Labels["tekton.dev/pipelineTask"]; ok {
		r.task = name
		return
	}

	if tr.Spec.TaskRef != nil {
		r.task = tr.Spec.TaskRef.Name
		return
	}

	r.task = fmt.Sprintf("Task %d", r.number)
}

func (r *Reader) readLiveTaskLogs(tr *v1.TaskRun) (<-chan Log, <-chan error, error) {
	defer func() {
		logger.Printf("PipelineRun Log readLiveTaskLogs end, task: %s\n", r.task)
	}()
	logger.Printf("PipelineRun Log readLiveTaskLogs start, task: %s\n", r.task)
	podC, podErrC, err := r.getTaskRunPodNames(tr)
	if err != nil {
		return nil, nil, err
	}
	logC, errC := r.readPodLogs(podC, podErrC, r.follow, r.timestamps)
	return logC, errC, nil
}

func (r *Reader) readAvailableTaskLogs(tr *v1.TaskRun) (<-chan Log, <-chan error, error) {
	defer func() {
		logger.Printf("PipelineRun Log readAvailableTaskLogs end, task: %s\n", r.task)
	}()
	logger.Printf("PipelineRun Log readAvailableTaskLogs start, task: %s\n", r.task)
	if !tr.HasStarted() {
		logger.Printf("PipelineRun Log readAvailableTaskLogs task %s has not started yet\n", r.task)
		return nil, nil, fmt.Errorf("task %s has not started yet", r.task)
	}

	// Check if taskrun failed on start up
	if err := hasTaskRunFailed(tr, r.task); err != nil {
		logger.Printf("PipelineRun Log readAvailableTaskLogs task %s failed: %s\n", r.task, err)
		if r.stream != nil {
			logger.Printf("PipelineRun Log readAvailableTaskLogs task %s failed: %s\n", r.task, err)
			fmt.Fprintf(r.stream.Err, "%s\n", err.Error())
		} else {
			logger.Printf("PipelineRun Log readAvailableTaskLogs task %s failed: %s\n", r.task, err)
			return nil, nil, err
		}
	}

	if tr.Status.PodName == "" {
		logger.Printf("PipelineRun Log readAvailableTaskLogs pod for taskrun %s not available yet\n", tr.Name)
		return nil, nil, fmt.Errorf("pod for taskrun %s not available yet", tr.Name)
	}

	podC := make(chan string)
	go func() {
		defer func() {
			logger.Printf("PipelineRun Log readAvailableTaskLogs go func defer end, task: %s\n", r.task)
		}()
		logger.Printf("PipelineRun Log readAvailableTaskLogs go func start, task: %s\n", r.task)
		defer close(podC)
		if tr.Status.PodName != "" {
			logger.Printf("PipelineRun Log readAvailableTaskLogs podC <- tr.Status.PodName, task: %s\n", r.task)
			if len(tr.Status.RetriesStatus) != 0 {
				for _, retryStatus := range tr.Status.RetriesStatus {
					podC <- retryStatus.PodName
				}
			}
			podC <- tr.Status.PodName
		}
	}()

	logC, errC := r.readPodLogs(podC, nil, false, r.timestamps)
	return logC, errC, nil
}

func (r *Reader) readStepsLogs(logC chan<- Log, errC chan<- error, steps []*step, pod *pods.Pod, follow, timestamps bool) {
	logger.Printf("PipelineRun Log readStepsLogs start, task: %s\n", r.task)
	defer func() {
		logger.Printf("PipelineRun Log readStepsLogs defer end, task: %s\n", r.task)
	}()
	for _, step := range steps {
		logger.Println("PipelineRun Log readStepsLogs start, " + step.name)
		if !follow && !step.hasStarted() {
			continue
		}

		container := pod.Container(step.container)
		// todo: 这个地方返回的是一个channel
		containerLogC, containerLogErrC, err := container.LogReader(follow, timestamps).Read()
		// todo: 如果获取日志的时候出错, 这个不会走
		if err != nil {
			errC <- fmt.Errorf("error in getting logs for step %s: %s", step.name, err)
			continue
		}

		for containerLogC != nil || containerLogErrC != nil {
			select {
			case l, ok := <-containerLogC:
				if !ok {
					containerLogC = nil
					logC <- Log{Task: r.task, TaskDisplayName: r.displayName, Step: step.name, Log: "EOFLOG"}
					continue
				}
				// todo: 实时写入日志,
				logC <- Log{Task: r.task, TaskDisplayName: r.displayName, Step: step.name, Log: l.Log}

			case e, ok := <-containerLogErrC:
				if !ok {
					containerLogErrC = nil
					continue
				}

				errC <- fmt.Errorf("failed to get logs for %s: %s", step.name, e)
			}
		}

		if err := container.Status(); err != nil {
			errC <- err
			return
		}
	}
}

func (r *Reader) readPodLogs(podC <-chan string, podErrC <-chan error, follow, timestamps bool) (<-chan Log, <-chan error) {
	logger.Println("PipelineRun Log readPodLogs start, task: " + r.task)
	defer func() {
		logger.Println("PipelineRun Log readPodLogs defer end, task: " + r.task)
	}()
	logC := make(chan Log)
	errC := make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		// todo: 这个地方可能有问题 ===========
		logger.Printf("PipelineRun Log readPodLogs go func defer start, task2: %s\n", r.task)
		defer func() {
			// todo: 这个地方没有执行到
			logger.Printf("PipelineRun Log readPodLogs go func defer end, task2: %s\n", r.task)
		}()

		// forward pod error to error stream
		if podErrC != nil {
			for podErr := range podErrC {
				logger.Printf("PipelineRun Log readPodLogs go func podErrC range, task: %s, error: %s\n", r.task, podErr)
				errC <- podErr
			}
		}
		wg.Done()

		// wait for all goroutines to close before closing errC channel
		logger.Println("PipelineRun Log readPodLogs Wait start")
		wg.Wait()
		logger.Println("PipelineRun Log readPodLogs Wait end")
		close(errC)
	}()

	// todo: 这个地方卡在了, 一直没有返回 ===========
	wg.Add(1)
	go func() {
		logger.Printf("PipelineRun Log readPodLogs go func start, task: %s\n", r.task)
		defer func() {
			logger.Printf("PipelineRun Log readPodLogs go func defer start, task: %s\n", r.task)
			close(logC)
			wg.Done()
		}()

		for podName := range podC {
			logger.Printf("PipelineRun Log readPodLogs go func range, podName: %s\n", podName)
			p := pods.New(podName, r.ns, r.clients.Kube, r.streamer)
			var pod *corev1.Pod
			var err error

			if follow {
				logger.Printf("PipelineRun Log readPodLogs go func Wait start, ....podName: %s\n", podName)
				pod, err = p.Wait()
				logger.Println("PipelineRun Log readPodLogs go func Wait end")
			} else {
				logger.Printf("PipelineRun Log readPodLogs go func Get start, ....podName: %s\n", podName)
				pod, err = p.Get()
				logger.Println("PipelineRun Log readPodLogs Get end")
			}
			if err != nil {
				errC <- fmt.Errorf("task %s failed: %s. Run tkn tr desc %s for more details", r.task, strings.TrimSpace(err.Error()), r.run)
			}
			steps := filterSteps(pod, r.allSteps, r.steps)
			r.readStepsLogs(logC, errC, steps, p, follow, timestamps)
			logger.Println("PipelineRun Log readPodLogs go func end")
		}
	}()

	logger.Println("PipelineRun Log readPodLogs end")
	return logC, errC
}

// Reading of logs should wait until the name of the pod is
// updated in the status. Open a watch channel on the task run
// and keep checking the status until the taskrun completes
// or the timeout is reached.
func (r *Reader) getTaskRunPodNames(run *v1.TaskRun) (<-chan string, <-chan error, error) {
	logger.Printf("PipelineRun Log getTaskRunPodNames start, task: %s\n", r.task)
	defer func() {
		logger.Printf("PipelineRun Log getTaskRunPodNames end, task: %s\n", r.task)
	}()
	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", r.run).String(),
	}

	watchRun, err := actions.Watch(taskrunGroupResource, r.clients, r.ns, opts)
	if err != nil {
		return nil, nil, err
	}

	podC := make(chan string)
	errC := make(chan error)

	go func() {
		defer func() {
			// todo: 这个地方有日志, 说明 channel 已经关闭了
			logger.Printf("PipelineRun Log getTaskRunPodNames go func defer end, task: %s\n", r.task)
			close(podC)
			close(errC)
			watchRun.Stop()
			logger.Printf("PipelineRun Log getTaskRunPodNames go func defer end222, task: %s\n", r.task)
		}()
		logger.Printf("PipelineRun Log getTaskRunPodNames go func start, task: %s\n", r.task)

		podMap := make(map[string]bool)
		addPod := func(name string) {
			if _, ok := podMap[name]; !ok {
				podMap[name] = true
				podC <- name
			}
		}

		if len(run.Status.RetriesStatus) != 0 {
			for _, retryStatus := range run.Status.RetriesStatus {
				addPod(retryStatus.PodName)
			}
		}
		if run.Status.PodName != "" {
			addPod(run.Status.PodName)
		}

		timeout := time.After(r.activityTimeout)

		for {
			select {
			case event := <-watchRun.ResultChan():
				var err error
				run, err = cast2taskrun(event.Object)
				if err != nil {
					errC <- err
					return
				}
				if run.Status.PodName != "" {
					addPod(run.Status.PodName)
					if !areRetriesScheduled(run, r.retries) {
						return
					}
				}
			case <-timeout:
				// Check if taskrun failed on start up
				if err := hasTaskRunFailed(run, r.task); err != nil {
					errC <- err
					return
				}
				// check if pod has been started and has a name
				if run.HasStarted() && run.Status.PodName != "" {
					if areRetriesScheduled(run, r.retries) {
						continue
					}
					return
				}
				errC <- fmt.Errorf("task %s has not started yet or pod for task not yet available", r.task)
				return
			}
		}
	}()

	return podC, errC, nil
}

func filterSteps(pod *corev1.Pod, allSteps bool, stepsGiven []string) []*step {
	logger.Printf("PipelineRun Log filterSteps start, allSteps: %v, stepsGiven: %v\n", allSteps, stepsGiven)
	defer func() {
		logger.Printf("PipelineRun Log filterSteps end, allSteps: %v, stepsGiven: %v\n", allSteps, stepsGiven)
	}()

	steps := []*step{}
	stepsInPod := getSteps(pod)
	logger.Printf("PipelineRun Log filterSteps stepsInPod: %+v\n", stepsInPod)
	for _, item := range stepsInPod {
		logger.Printf("PipelineRun Log filterSteps stepsInPod item: %+v\n", item)
	}

	if allSteps {
		steps = append(steps, getInitSteps(pod)...)
	}

	if len(stepsGiven) == 0 {
		steps = append(steps, stepsInPod...)
		return steps
	}

	stepsToAdd := map[string]bool{}
	for _, s := range stepsGiven {
		stepsToAdd[s] = true
	}

	for _, sp := range stepsInPod {
		if stepsToAdd[sp.name] {
			steps = append(steps, sp)
		}
	}

	return steps
}

func getInitSteps(pod *corev1.Pod) []*step {
	status := map[string]corev1.ContainerState{}
	for _, ics := range pod.Status.InitContainerStatuses {
		status[ics.Name] = ics.State
	}

	steps := []*step{}
	for _, ic := range pod.Spec.InitContainers {
		steps = append(steps, &step{
			name:      strings.TrimPrefix(ic.Name, "step-"),
			container: ic.Name,
			state:     status[ic.Name],
		})
	}

	return steps
}

func getSteps(pod *corev1.Pod) []*step {
	status := map[string]corev1.ContainerState{}
	for _, cs := range pod.Status.ContainerStatuses {
		status[cs.Name] = cs.State
	}

	steps := []*step{}
	for _, c := range pod.Spec.Containers {
		steps = append(steps, &step{
			name:      strings.TrimPrefix(c.Name, "step-"),
			container: c.Name,
			state:     status[c.Name],
		})
	}

	return steps
}

func hasTaskRunFailed(tr *v1.TaskRun, taskName string) error {
	if isFailure(tr) {
		return fmt.Errorf("task %s has failed: %s", taskName, tr.Status.Conditions[0].Message)
	}
	return nil
}

func cast2taskrun(obj runtime.Object) (*v1.TaskRun, error) {
	var run *v1.TaskRun
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct, &run); err != nil {
		return nil, err
	}
	return run, nil
}

func isFailure(tr *v1.TaskRun) bool {
	conditions := tr.Status.Conditions
	return len(conditions) != 0 && conditions[0].Status == corev1.ConditionFalse
}

func areRetriesScheduled(tr *v1.TaskRun, retries int) bool {
	if tr.IsDone() {
		return false
	}
	retriesDone := len(tr.Status.RetriesStatus)
	return retriesDone < retries
}
