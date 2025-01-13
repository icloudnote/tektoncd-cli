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

package pods

import (
	"context"
	"fmt"
	"io"
	logger "log"
	"sync"
	"time"

	"github.com/tektoncd/cli/pkg/pods/stream"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	k8s "k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Stream struct {
	name string
	pods typedv1.PodInterface
	opts *corev1.PodLogOptions
}

func NewStream(pods typedv1.PodInterface, name string, opts *corev1.PodLogOptions) stream.Streamer {
	return &Stream{name, pods, opts}
}

// Stream Creates a stream object which allows reading the logs
func (s *Stream) Stream() (io.ReadCloser, error) {
	return s.pods.GetLogs(s.name, s.opts).Stream(context.Background())
}

type Pod struct {
	Name     string
	Ns       string
	Kc       k8s.Interface
	Streamer stream.NewStreamerFunc
}

func New(name, ns string, client k8s.Interface, streamer stream.NewStreamerFunc) *Pod {
	return &Pod{
		Name: name, Ns: ns,
		Kc:       client,
		Streamer: streamer,
	}
}

func NewWithDefaults(name, ns string, client k8s.Interface) *Pod {
	return &Pod{
		Name: name, Ns: ns,
		Kc:       client,
		Streamer: NewStream,
	}
}

// Wait wait for the pod to get up and running
func (p *Pod) Wait() (*corev1.Pod, error) {
	logger.Println("PipelineRun Log Wait start")
	// ensure pod exists before we actually check for it
	if _, err := p.Get(); err != nil {
		return nil, err
	}

	stopC := make(chan struct{})
	eventC := make(chan interface{})
	mu := sync.Mutex{}
	defer func() {
		mu.Lock()
		close(stopC)
		close(eventC)
		mu.Unlock()
	}()

	logger.Println("PipelineRun Log Wait watcher start")
	p.watcher(stopC, eventC, &mu)
	logger.Println("PipelineRun Log Wait watcher end")

	var pod *corev1.Pod
	var err error
	for e := range eventC {
		pod, err = checkPodStatus(e)
		if pod != nil || err != nil {
			break
		}
	}
	logger.Println("PipelineRun Log Wait end")
	return pod, err
}

func (p *Pod) watcher(stopC <-chan struct{}, eventC chan<- interface{}, mu *sync.Mutex) {
	logger.Println("PipelineRun Log watcher")
	factory := informers.NewSharedInformerFactoryWithOptions(
		p.Kc, time.Second*10,
		informers.WithNamespace(p.Ns),
		informers.WithTweakListOptions(podOpts(p.Name)))

	_, err := factory.Core().V1().Pods().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stopC:
					return
				default:
					// default is used to avoid pseudo-random selection of multiple matching cases
					logger.Printf("Pod added: %s", obj.(*corev1.Pod).Name)
					eventC <- obj
				}
			},
			UpdateFunc: func(_, newObj interface{}) {
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stopC:
					return
				default:
					logger.Printf("Pod update: %s", newObj.(*corev1.Pod).Name)
					logger.Printf("Pod update: %+v", newObj.(*corev1.Pod).Status.Phase)
					eventC <- newObj
				}
			},
			DeleteFunc: func(obj interface{}) {
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stopC:
					return
				default:
					logger.Printf("Pod delete: %s", obj.(*corev1.Pod).Name)
					eventC <- obj
				}
			},
		})
	if err != nil {
		return
	}

	logger.Println("PipelineRun Log watcher start")
	factory.Start(stopC)
	logger.Println("PipelineRun Log watcher start end")
	factory.WaitForCacheSync(stopC)
	logger.Println("PipelineRun Log watcher WaitForCacheSync end")
}

func podOpts(name string) func(opts *metav1.ListOptions) {
	return func(opts *metav1.ListOptions) {
		opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
	}
}

func checkPodStatus(obj interface{}) (*corev1.Pod, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("failed to cast to pod object")
	}

	if pod.DeletionTimestamp != nil {
		return pod, fmt.Errorf("failed to run the pod %s ", pod.Name)
	}

	logger.Printf("Pod status: %s", pod.Status.Phase)
	if pod.Status.Phase == corev1.PodSucceeded ||
		pod.Status.Phase == corev1.PodRunning ||
		pod.Status.Phase == corev1.PodFailed {
		return pod, nil
	}

	// Handle any issues with pulling images that may fail
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodInitialized || c.Type == corev1.ContainersReady {
			if c.Status == corev1.ConditionUnknown {
				return pod, fmt.Errorf("%s", c.Message)
			}
		}
	}

	return nil, nil
}

// Get gets the pod
func (p *Pod) Get() (*corev1.Pod, error) {
	return p.Kc.CoreV1().Pods(p.Ns).Get(context.Background(), p.Name, metav1.GetOptions{})
}

// Container returns the an instance of Container
func (p *Pod) Container(c string) *Container {
	return &Container{
		name:        c,
		pod:         p,
		NewStreamer: p.Streamer,
	}
}

// Stream returns the stream object for given container and mode
// in order to fetch the logs
func (p *Pod) Stream(opt *corev1.PodLogOptions) (io.ReadCloser, error) {
	pods := p.Kc.CoreV1().Pods(p.Ns)
	if pods == nil {
		return nil, fmt.Errorf("error getting pods")
	}

	return p.Streamer(pods, p.Name, opt).Stream()
}
