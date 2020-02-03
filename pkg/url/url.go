package url

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pkg/errors"
	applabels "github.com/redhat-developer/odo-fork/pkg/application/labels"
	componentlabels "github.com/redhat-developer/odo-fork/pkg/component/labels"
	"github.com/redhat-developer/odo-fork/pkg/kclient"
	urlLabels "github.com/redhat-developer/odo-fork/pkg/url/labels"
	"github.com/redhat-developer/odo-fork/pkg/util"
	iextensionsv1 "k8s.io/api/extensions/v1beta1"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
)

// Get returns URL defination for given URL name
// func (urls UrlList) Get(urlName string) Url {
// 	for _, url := range urls.Items {
// 		if url.Name == urlName {
// 			return url
// 		}
// 	}
// 	return Url{}

// }

// func (ingresses iextensionsv1.IngressList) Get(urlName string) iextensionsv1.Ingress {
// 	for _, ingress := range ingresses.Items {
// 		if ingress.Name == urlName {
// 			return ingress
// 		}
// 	}
// 	return iextensionsv1.Ingress{}

// }

// Delete deletes a URL
func Delete(client *kclient.Client, urlName string, applicationName string) error {

	// Namespace the URL name
	namespaceKubernetesObject, err := util.NamespaceKubernetesObject(urlName, applicationName)
	if err != nil {
		return errors.Wrapf(err, "unable to create namespaced name")
	}

	return client.DeleteIngress(namespaceKubernetesObject)
}

// Create creates a URL and returns url string and error if any
// portNumber is the target port number for the ingress and is -1 in case no port number is specified in which case it is automatically detected for components which expose only one service port)
func Create(client *kclient.Client, urlName string, portNumber int, ingressDomain string, https bool, componentName string, applicationName string) (string, error) {
	labels := urlLabels.GetLabels(urlName, componentName, applicationName, false)
	serviceName, err := util.NamespaceKubernetesObject(componentName, applicationName)
	if err != nil {
		return "", errors.Wrapf(err, "unable to create namespaced name")
	}

	urlName, err = util.NamespaceKubernetesObject(urlName, applicationName)
	if err != nil {
		return "", errors.Wrapf(err, "unable to create namespaced name")
	}
	secretName := ""
	if https == true {
		// generate SSl certificate
		fmt.Printf("Https is true, creating SSL certificate.")
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			fmt.Printf("unale to generate rsa key ")
			fmt.Println(errors.Cause(err))
			return "", errors.Wrap(err, "unable to generate rsa key")
		}
		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName:   "Udo self-signed certificate",
				Organization: []string{"Udo"},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().Add(time.Hour * 24 * 365),
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		certificateDerEncoding, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
		if err != nil {
			fmt.Printf("unable to create certificate ")
			fmt.Println(errors.Cause(err))
			return "", errors.Wrap(err, "unable to create certificate")
		}
		out := &bytes.Buffer{}
		pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDerEncoding})
		certPemEncode := out.String()
		certPemByteArr := []byte(certPemEncode)

		tlsPrivKeyEncoding := x509.MarshalPKCS1PrivateKey(privateKey)
		pem.Encode(out, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: tlsPrivKeyEncoding})
		keyPemEncode := out.String()
		keyPemByteArr := []byte(keyPemEncode)

		// create tls secret
		secret, err := client.CreateTLSSecret(certPemByteArr, keyPemByteArr, componentName, applicationName, portNumber)
		if err != nil {
			fmt.Printf("unable to create tls secret ")
			fmt.Println(errors.Cause(err))
			return "", errors.Wrap(err, "unable to create tls secret: "+secret.Name)
		}
		secretName = secret.Name

	}
	ingressParam := kclient.IngressParamater{urlName, serviceName, ingressDomain, intstr.FromInt(portNumber), secretName}
	// Pass in the namespace name, link to the service (componentName) and labels to create a ingress
	ingress, err := client.CreateIngress(ingressParam, labels)
	if err != nil {
		return "", errors.Wrap(err, "unable to create ingress")
	}

	return GetURLString(GetProtocol(*ingress), ingressDomain), nil
}

// List lists the URLs in an application. The results can further be narrowed
// down if a component name is provided, which will only list URLs for the
// given component
func List(client *kclient.Client, componentName string, applicationName string) (iextensionsv1.IngressList, error) {

	labelSelector := fmt.Sprintf("%v=%v", applabels.ApplicationLabel, applicationName)

	if componentName != "" {
		labelSelector = labelSelector + fmt.Sprintf(",%v=%v", componentlabels.ComponentLabel, componentName)
	}

	glog.V(4).Infof("Listing ingresses with label selector: %v", labelSelector)
	ingresses, err := client.ListIngresses(labelSelector)
	if err != nil {
		return iextensionsv1.IngressList{}, errors.Wrap(err, "unable to list ingress names")
	}

	var urls []iextensionsv1.Ingress
	for _, i := range ingresses {
		a := getMachineReadableFormat(i)
		urls = append(urls, a)
	}

	urlList := getMachineReadableFormatForList(urls)
	return urlList, nil
}

func GetProtocol(ingress iextensionsv1.Ingress) string {
	if ingress.Spec.TLS != nil {
		return "https"
	}
	return "http"

}

// GetURLString returns a string representation of given url
func GetURLString(protocol, URL string) string {
	return protocol + "://" + URL
}

// Exists checks if the url exists in the component or not
// urlName is the name of the url for checking
// componentName is the name of the component to which the url's existence is checked
// applicationName is the name of the application to which the url's existence is checked
func Exists(client *kclient.Client, urlName string, componentName string, applicationName string) (bool, error) {
	urls, err := List(client, componentName, applicationName)
	if err != nil {
		return false, errors.Wrap(err, "unable to list the urls")
	}

	for _, url := range urls.Items {
		if url.Name == urlName {
			return true, nil
		}
	}
	return false, nil
}

// GetComponentServicePortNumbers returns the port numbers exposed by the service of the component
// componentName is the name of the component
// applicationName is the name of the application
func GetComponentServicePortNumbers(client *kclient.Client, componentName string, applicationName string) ([]int, error) {
	componentLabels := componentlabels.GetLabels(componentName, applicationName, false)
	componentSelector := util.ConvertLabelsToSelector(componentLabels)

	services, err := client.GetServicesFromSelector(componentSelector)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the service")
	}

	var ports []int

	for _, service := range services {
		for _, port := range service.Spec.Ports {
			ports = append(ports, int(port.Port))
		}
	}

	return ports, nil
}

// GetURLName returns a url name from the component name and the given port number
func GetURLName(componentName string, componentPort int) string {
	if componentPort == -1 {
		return componentName
	}
	return fmt.Sprintf("%v-%v", componentName, componentPort)
}

// GetValidPortNumber checks if the given port number is a valid component port or not
// if port number is not provided and the component is a single port component, the component port is returned
// port number is -1 if the user does not specify any port
func GetValidPortNumber(client *kclient.Client, portNumber int, componentName string, applicationName string) (int, error) {
	componentPorts, err := GetComponentServicePortNumbers(client, componentName, applicationName)
	if err != nil {
		return portNumber, errors.Wrapf(err, "unable to get exposed ports for component %s", componentName)
	}

	// port number will be -1 if the user doesn't specify any port
	if portNumber == -1 {

		switch {
		case len(componentPorts) > 1:
			return portNumber, errors.Errorf("port for the component %s is required as it exposes %d ports: %s", componentName, len(componentPorts), strings.Trim(strings.Replace(fmt.Sprint(componentPorts), " ", ",", -1), "[]"))
		case len(componentPorts) == 1:
			return componentPorts[0], nil
		default:
			return portNumber, errors.Errorf("no port is exposed by the component %s", componentName)
		}
	} else {
		for _, port := range componentPorts {
			if portNumber == port {
				return portNumber, nil
			}
		}
	}

	return portNumber, nil
}

func getMachineReadableFormat(i iextensionsv1.Ingress) iextensionsv1.Ingress {
	return iextensionsv1.Ingress{
		TypeMeta:   metav1.TypeMeta{Kind: "Ingress", APIVersion: "extensions/v1beta1"},
		ObjectMeta: metav1.ObjectMeta{Name: i.Labels[urlLabels.URLLabel]},
		Spec:       iextensionsv1.IngressSpec{TLS: i.Spec.TLS, Rules: i.Spec.Rules},
	}

}

func getMachineReadableFormatForList(ingresses []iextensionsv1.Ingress) iextensionsv1.IngressList {
	return iextensionsv1.IngressList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "udo.udo.io/v1alpha1",
		},
		ListMeta: metav1.ListMeta{},
		Items:    ingresses,
	}
}
