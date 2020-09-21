package hope

import (
	"fmt"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

type JobStatus int

const (
	JobStatusUnknown JobStatus = 0
	JobStatusRunning = 1
	JobStatusComplete = 2
	JobStatusFailed = 3
)

// Check to see if the provided job has completed, or is still running.
func GetJobStatus(log *logrus.Entry, kubectl *kubeutil.Kubectl, job string) (JobStatus, error) {
	output, err := kubeutil.GetKubectl(kubectl, "get", "job", job, "-o", "template={{range .status.conditions}}{{.type}}{{end}}")
	if err != nil {
		return JobStatusUnknown, err
	}

	switch output {
	case "Complete":
		return JobStatusComplete, nil
	case "Failed":
		return JobStatusFailed, nil
	default:
		return JobStatusRunning, nil		
	}
}

func AttachToLogsIfContainersRunning(kubectl *kubeutil.Kubectl, job string) error {
	jobSelector := fmt.Sprintf("job-name=%s", job)
	
	// Wait for this loop to finish without failure.
	// Exponential backoff for re-attempts, max 12 seconds
	return kubeutil.ExecKubectl(kubectl, "logs", "-f", "-l", jobSelector)
}

func GetPodsForJob(kubectl *kubeutil.Kubectl, job string) (*[]string, error) {
	jobSelector := fmt.Sprintf("job-name=%s", job)
	output, err := kubeutil.GetKubectl(kubectl, "get", "pods", "-l", jobSelector, "-o", "template={{range .items}}{{.metadata.name}}{{end}}")
	if err != nil {
		return nil, err
	}

	pods := strings.Split(output, "\n")
	return &pods, nil
}
