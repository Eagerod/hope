package cmd

import (
	"errors"
	"fmt"
	"net/url"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/sliceutil"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a node within the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]

		// URL parsing is a bit better at identifying parameters if there's a
		//   protocol on the string passed in, so fake in ssh as the protocol to
		//   help it parse a little more reliably.
		host_url, err := url.Parse(fmt.Sprintf("ssh://%s", host))
		if err != nil {
			return err
		}

		log.Info("Bootstrapping a node...")

		podNetworkCidr := viper.GetString("pod_network_cidr")
		masters := viper.GetStringSlice("masters")
		masterLoadBalancer := viper.GetString("master_load_balancer")

		isMaster := sliceutil.StringInSlice(host, masters)
		isWorker := sliceutil.StringInSlice(host, viper.GetStringSlice("nodes"))
		isLoadBalancer := masterLoadBalancer == host

		if isMaster && isLoadBalancer {
			return errors.New(fmt.Sprintf("Host %s cannot be master and load balancer.", host))
		}

		if isWorker && isLoadBalancer {
			return errors.New(fmt.Sprintf("Host %s cannot be worker and load balancer.", host))
		}

		if isLoadBalancer {
			return hope.InitLoadBalancer(log.WithFields(log.Fields{}), host)
		}

		if isMaster && isWorker {
			log.Info("Node ", host, " appears in both master and node configurations. Creating master and removing NoSchedule taint...")

			if err := hope.CreateClusterMaster(log.WithFields(log.Fields{}), host, podNetworkCidr); err != nil {
				return err
			}

			kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
			if err != nil {
				return err
			}

			defer kubectl.Destroy()

			if err := hope.TaintNodeByHost(kubectl, host_url.Host, "node-role.kubernetes.io/master:NoSchedule-"); err != nil {
				return err
			}
		} else if isMaster {
			return hope.CreateClusterMaster(log.WithFields(log.Fields{}), host, podNetworkCidr)
		} else if isWorker {
			// Have to send in a master ip for it to grab a join token.
			aMaster := masters[0]

			if err := hope.CreateClusterNode(log.WithFields(log.Fields{}), host, aMaster); err != nil {
				return err
			}
		}

		return errors.New(fmt.Sprintf("Failed to find node %s in config", host))
	},
}
