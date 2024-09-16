package k8s

import (
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func OpenPortForwardPodWrapper(
	clientset *kubernetes.Clientset,
	restConfig *rest.Config,
	podName string,
	namespace string,
	podPort int,
	podLocalPort int,
	stopChannel chan struct{},
) error {
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	portForwardRequest := PortForwardAPodRequest{
		RestConfig: restConfig,
		Pod: v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
		},
		PodPort:   podPort,
		LocalPort: podLocalPort,
		StopCh:    stopChannel,
		ReadyCh:   readyCh,
	}

	// Check to see if the port is already used
	err := CheckForExistingPortForwards(podLocalPort)
	if err != nil {
		log.Error().Msgf("unable to start port forward for pod %s in namespace %s: %s", podName, namespace, err)
		return err
	}

	go func() {
		err := PortForwardPodWithRetry(clientset, portForwardRequest)
		if err != nil {
			log.Error().Err(err).Msg(err.Error())
		}
	}()

	select {
	case <-stopChannel:
		log.Info().Msg("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		log.Info().Msg("port forwarding is ready to get traffic")
	}

	log.Info().Msgf("pod %q at namespace %q has port-forward accepting local connections at port %d", podName, namespace, podLocalPort)
	return nil
}
