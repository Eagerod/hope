package hope

import (
	"fmt"
	"math"
	"strings"
	"time"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

type JobStatus int

const (
	JobStatusUnknown JobStatus = iota
	JobStatusRunning
	JobStatusComplete
	JobStatusFailed
)

// Check to see if the provided job has completed, or is still running.
func GetJobStatus(log *logrus.Entry, kubectl *kubeutil.Kubectl, namespace, job string) (JobStatus, error) {
	output, err := kubeutil.GetKubectl(kubectl, "get", "-n", namespace, "job", job, "-o", "template={{range .status.conditions}}{{.type}}{{end}}")
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

func FollowLogsIfContainersRunning(kubectl *kubeutil.Kubectl, namespace, job string) error {
	jobSelector := fmt.Sprintf("job-name=%s", job)
	return kubeutil.ExecKubectl(kubectl, "logs", "-n", namespace, "-f", "-l", jobSelector)
}

func GetPodsForJob(kubectl *kubeutil.Kubectl, namespace, job string) (*[]string, error) {
	jobSelector := fmt.Sprintf("job-name=%s", job)
	output, err := kubeutil.GetKubectl(kubectl, "get", "-n", namespace, "pods", "-l", jobSelector, "-o", "template={{range .items}}{{.metadata.name}} {{end}}")
	if err != nil {
		return nil, err
	}

	pods := strings.Split(strings.TrimSpace(output), " ")
	return &pods, nil
}

func FollowLogsAndPollUntilJobComplete(log *logrus.Entry, kubectl *kubeutil.Kubectl, nsJob string, maxAttempts int, failedPollDelayMaxSeconds int) error {
	namespace := "default"
	job := nsJob

	components := strings.Split(nsJob, "/")
	if len(components) == 2 {
		namespace = components[0]
		job = components[1]
	}

	// Check the job status before anything.
	// It's possible that the job ran long ago, and pods have been cleaned up.
	// If that's the case, attempting to attach to logs will fail; and that
	//   won't be straight-forward to recover from.
	status, err := GetJobStatus(log, kubectl, namespace, job)
	if err != nil {
		return err
	}

	switch status {
	case JobStatusFailed:
		return fmt.Errorf("job %s failed", job)
	case JobStatusComplete:
		log.Debug("Job ", nsJob, " successful.")
		return nil
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		attemptsDuration := math.Pow(2, float64(attempt))
		onFailureSleepSeconds := int(math.Min(attemptsDuration, float64(failedPollDelayMaxSeconds)))

		logsErr := FollowLogsIfContainersRunning(kubectl, namespace, job)
		if logsErr != nil {
			log.Warn(logsErr)
		}

		// Logs may have successfully attached and printed for failed
		//   containers, so just because the log function succeeded, we
		//   can't assume success.
		status, err := GetJobStatus(log, kubectl, namespace, job)
		if err != nil {
			return err
		}

		switch status {
		case JobStatusFailed:
			return fmt.Errorf("job %s failed", job)
		case JobStatusComplete:
			log.Debug("Job ", job, " successful.")
			return nil
		}

		if onFailureSleepSeconds == failedPollDelayMaxSeconds {
			log.Debug("Checking pod events for details...")
			// Check the event log for the pods associated with this job.
			// There may be something useful in there.
			pods, err := GetPodsForJob(kubectl, namespace, job)
			if err != nil {
				log.Warn(err)
			} else {
				// TODO: Keep track of which pods have been printed, and if
				//   there have been no events for a given pod since the last
				//   time we tried to print them, don't print anything.
				for _, pods := range *pods {
					involvedObject := fmt.Sprintf("involvedObject.name=%s", pods)
					kubeutil.ExecKubectl(kubectl, "get", "events", "--field-selector", involvedObject)
				}
			}
		}
		if logsErr != nil {
			log.Warn("Failed to fetch logs for job ", job, ". Waiting ", onFailureSleepSeconds, " seconds and trying again.")
		} else {
			log.Warn("Logs fetched, but job ", job, " is still running. Waiting ", onFailureSleepSeconds, " seconds and trying again.")
		}
		time.Sleep(time.Second * time.Duration(onFailureSleepSeconds))
	}

	return fmt.Errorf("job did not finish within %d attempts. The job may still be running", maxAttempts)
}
