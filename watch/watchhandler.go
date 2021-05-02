package watch

import (
	"container/list"
	"flag"
	"log"
	"os"
	"path/filepath"

	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// WatchHandler -
type WatchHandler struct {
	extensionsClient apixv1beta1client.ApiextensionsV1beta1Interface
	RestAPIClient    kubernetes.Interface
	WebSocketHandle  *WebSocketHandler
	// cluster info
	clusterAPIServerVersion *version.Info
	cloudVendor             string
	// pods list
	pdm map[int]*list.List
	// node list
	ndm map[int]*list.List
	// services list
	sdm map[int]*list.List
	// secrets list
	secretdm               map[int]*list.List
	jsonReport             jsonFormat
	informNewDataChannel   chan int
	aggregateFirstDataFlag bool
}

// GetAggregateFirstDataFlag return pointer
func (wh *WatchHandler) GetAggregateFirstDataFlag() *bool {
	return &wh.aggregateFirstDataFlag
}

//CreateWatchHandler -
func CreateWatchHandler() *WatchHandler {

	config := parseArgument()
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Print(err.Error())
		return nil
	}
	extensionsClientSet, err := apixv1beta1client.NewForConfig(config)

	if err != nil {
		log.Print(err.Error())
		return nil
	}

	var clusterName string
	if clusterName = os.Getenv("CA_CLUSTER_NAME"); clusterName == "" {
		log.Println("there is no cluster name environment variable, envKey:CA_CLUSTER_NAME")
		//clusterName = "superCluster"
		return nil
	}

	var reportURL string
	if reportURL = os.Getenv("CA_K8S_REPORT_URL"); reportURL == "" {
		log.Println("there is no report url environment variable, envKey:CA_K8S_REPORT_URL")
		//reportURL = "report.eudev2.cyberarmorsoft.com"
		return nil
	}

	var cusGUID string
	if cusGUID = os.Getenv("CA_CUSTOMER_GUID"); cusGUID == "" {
		log.Println("there is no customer guid environment variable, envKey:CA_CUSTOMER_GUID")
		//cusGUID = "1e3a88bf-92ce-44f8-914e-cbe71830d566"
		return nil
	}

	result := WatchHandler{RestAPIClient: clientset,
		WebSocketHandle:  createWebSocketHandler(reportURL, "k8s/cluster-reports", clusterName, cusGUID),
		extensionsClient: extensionsClientSet,
		pdm:              make(map[int]*list.List), ndm: make(map[int]*list.List), sdm: make(map[int]*list.List), secretdm: make(map[int]*list.List),
		jsonReport: jsonFormat{Nodes: ObjectData{}, Services: ObjectData{}, MicroServices: ObjectData{},
			Pods: ObjectData{}}, informNewDataChannel: make(chan int), aggregateFirstDataFlag: true}

	return &result
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
func parseArgument() *restclient.Config {
	var kubeconfigpath *string
	var config *restclient.Config
	var err error

	home := homeDir()
	configtype := flag.Int("configtype", 0, "newForConfig = 0, inClusterConfig = 1")
	if len(os.Args) < 3 && home != "" {
		kubeconfigpath = flag.String("kubeconfigpath", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfigpath = flag.String("kubeconfigpath", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	switch *configtype {
	case 0:
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfigpath)
		if err != nil {
			log.Printf("kubeconfig path is %s\n", *kubeconfigpath)
			log.Print(err.Error())
			return nil
		}
	case 1:
		config, err = restclient.InClusterConfig()
		if err != nil {
			log.Print(err.Error())
			return nil
		}
	}

	return config
}

// SetFirstReportFlag set first report flag
func (wh *WatchHandler) SetFirstReportFlag(first bool) {
	wh.jsonReport.FirstReport = first
}

// GetFirstReportFlag get first report flag
func (wh *WatchHandler) GetFirstReportFlag() bool {
	return wh.jsonReport.FirstReport
}
