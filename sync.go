package main

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"log"
	"net/http"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func gitSyncWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /api/git-sync-webhook request")
	// 1. find all pods which has `git-sync` container
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	var errors []error
	podNames, err := getGitSyncPods(clientset)
	if err != nil {
		errors = append(errors, err)
	} else {
		// 2. exec `kill -HUP 1` on those containers
		for _, podName := range podNames {
			err = sendSignalToGitSync(config, clientset, podName, "HUP")
			if err != nil {
				log.Printf("Failed to send signal to pod %s: %v\n", podName, err)
				errors = append(errors, err)
			}
		}
	}
	if r.URL.Query().Get("webserver") != "" {
		err = reloadWebserver(clientset)
		if err != nil {
			errors = append(errors, err)
		}
	}
	result, err := json.Marshal(errors)
	if err != nil {
		log.Printf("Failed to marshal errors: %v\n", err)
	}
	w.Header().Set("Content-Type", "application/json")
	if len(errors) == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, err = fmt.Fprint(w, string(result))
	if err != nil {
		log.Printf("Failed to write http response: %v\n", err)
	}
}

func reloadWebserver(clientset *kubernetes.Clientset) error {
	deploymentsClient := clientset.AppsV1().Deployments("airflow")
	data := fmt.Sprintf(`{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`, time.Now().Format("20231027160454"))
	ctx := context.Background()
	deployments, err := deploymentsClient.List(ctx, metav1.ListOptions{LabelSelector: "component=webserver"})
	if err != nil {
		return err
	}
	for _, deploy := range deployments.Items {
		_, err := deploymentsClient.Patch(ctx, deploy.Name, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func sendSignalToGitSync(config *rest.Config, clientset *kubernetes.Clientset, podName string, signal string) error {
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace("airflow").SubResource("exec")
	option := &v1.PodExecOptions{
		Container: "git-sync",
		Command:   []string{"sh", "-c", fmt.Sprintf("kill -%s 1", signal)},
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return err
	}
	return nil
}

func getGitSyncPods(clientset *kubernetes.Clientset) ([]string, error) {
	var podsList []string
	pods, err := clientset.CoreV1().Pods("airflow").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return podsList, err
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if container.Name == "git-sync" {
				podsList = append(podsList, pod.Name)
			}
		}
	}
	return podsList, nil
}
