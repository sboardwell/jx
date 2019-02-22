package cmd

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultFlaggerNamespace             = "istio-system"
	defaultFlaggerReleaseName           = kube.DefaultFlaggerReleaseName
	defaultFlaggerVersion               = ""
	defaultFlaggerRepo                  = "https://flagger.app"
	optionGrafanaChart                  = "grafana-chart"
	defaultFlaggerProductionEnvironment = "production"
	defaultIstioGateway                 = "jx-gateway"
)

var (
	createAddonFlaggerLong = templates.LongDesc(`
		Creates the Flagger addon
`)

	createAddonFlaggerExample = templates.Examples(`
		# Create the Flagger addon
		jx create addon flagger
	`)
)

type CreateAddonFlaggerOptions struct {
	CreateAddonOptions
	Chart                 string
	GrafanaChart          string
	ProductionEnvironment string
	IstioGateway          string
}

func NewCmdCreateAddonFlagger(commonOpts *CommonOptions) *cobra.Command {
	options := &CreateAddonFlaggerOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "flagger",
		Short:   "Create the Flagger addon for Canary deployments",
		Long:    createAddonFlaggerLong,
		Example: createAddonFlaggerExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addFlags(cmd, defaultFlaggerNamespace, defaultFlaggerReleaseName, defaultFlaggerVersion)

	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartFlagger, "The name of the chart to use")
	cmd.Flags().StringVarP(&options.GrafanaChart, optionGrafanaChart, "", kube.ChartFlaggerGrafana, "The name of the Flagger Grafana chart to use")
	cmd.Flags().StringVarP(&options.ProductionEnvironment, "environment", "e", defaultFlaggerProductionEnvironment, "The name of the production environment where Istio will be enabled")
	cmd.Flags().StringVarP(&options.IstioGateway, "istio-gateway", "", defaultIstioGateway, "The name of the Istio Gateway that will be created if it does not exist")
	return cmd
}

// Create the addon
func (o *CreateAddonFlaggerOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	if o.GrafanaChart == "" {
		return util.MissingOption(optionGrafanaChart)
	}
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that Helm is present")
	}

	values := []string{}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	err = o.addHelmRepoIfMissing(defaultFlaggerRepo, "flagger", "", "")
	if err != nil {
		return errors.Wrap(err, "Flagger deployment failed")
	}
	err = o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values, nil, "")
	if err != nil {
		return errors.Wrap(err, "Flagger deployment failed")
	}
	err = o.installChart(o.ReleaseName+"-grafana", o.GrafanaChart, o.Version, o.Namespace, true, values, nil, "")
	if err != nil {
		return errors.Wrap(err, "Flagger Grafana deployment failed")
	}

	// Enable Istio in production namespace
	if o.ProductionEnvironment != "" {
		client, err := o.KubeClient()
		if err != nil {
			return errors.Wrap(err, "error enabling Istio in production namespace")
		}
		var ns string
		ns, err = o.findEnvironmentNamespace(o.ProductionEnvironment)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error enabling Istio for environment %s", o.ProductionEnvironment))
		}
		log.Infof("Enabling Istio in namespace %s\n", ns)
		patch := []byte(`{"metadata":{"labels":{"istio-injection":"enabled"}}}`)
		_, err = client.CoreV1().Namespaces().Patch(ns, types.MergePatchType, patch)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error enabling Istio in namespace %s", ns))
		}
	}

	// Create the Istio gateway
	if o.IstioGateway != "" {
		istioClient, err := o.CreateAddonOptions.CommonOptions.IstioClient()
		if err != nil {
			return errors.Wrap(err, "error building Istio client")
		}
		gateway, err := istioClient.NetworkingV1alpha3().Gateways(defaultIstioNamespace).Get(o.IstioGateway, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			gateway = &istiov1alpha3.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      o.IstioGateway,
					Namespace: defaultIstioNamespace,
				},
				Spec: istiov1alpha3.GatewaySpec{
					Selector: map[string]string{"istio": "ingressgateway"},
					Servers: []istiov1alpha3.Server{
						// TODO add https port if enabled
						{
							Port: istiov1alpha3.Port{
								Number:   80,
								Name:     "http",
								Protocol: "HTTP",
							},
							Hosts: []string{"*"},
						},
					},
				},
			}

			log.Infof("Creating Istio gateway: %s\n", o.IstioGateway)
			gateway, err = istioClient.NetworkingV1alpha3().Gateways(defaultIstioNamespace).Create(gateway)
			if err != nil {
				return errors.Wrap(err, "error creating Istio gateway")
			}
		} else {
			log.Infof("Istio gateway already exists: %s\n", o.IstioGateway)
		}
	}
	return nil
}
