/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	controllers "github.com/mittwald/harbor-operator/controllers/registries"
	"github.com/mittwald/harbor-operator/controllers/registries/config"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	log      = logf.Log.WithName("cmd")
)

func init() {

	utilruntime.Must(registriesv1alpha2.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	pflag.String(config.FlagMetricsAddress, ":8080", "The address the metric endpoint binds to.")
	pflag.Bool(config.FlagEnableLeaderElection, false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.String(config.FlagHelmClientRepoCachePath,
		"/tmp/.helmcache", "helm client repository cache path")
	pflag.String(config.FlagHelmClientRepoConfPath,
		"/tmp/.helmconfig", "helm client repository config path")

	pflag.Parse()

	viper.SetEnvPrefix("HARBOR_OPERATOR")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Error(err, "failed parsing pflag CommandLine")
		os.Exit(1)
	}

	config.FromViper()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: config.MetricsAddr + ":9443",
		},
		LeaderElection:   config.EnableLeaderElection,
		LeaderElectionID: "a1e7caa2.mittwald.de",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.InstanceChartRepositoryReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("registries").WithName("InstanceChartRepository"),
		Scheme:             mgr.GetScheme(),
		HelmClientReceiver: AddHelmClientReceiver(mgr),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InstanceChartRepository")
		os.Exit(1)
	}
	if err = (&controllers.InstanceReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("registries").WithName("Instance"),
		Scheme:             mgr.GetScheme(),
		HelmClientReceiver: AddHelmClientReceiver(mgr),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Instance")
		os.Exit(1)
	}
	if err = (&controllers.RegistryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("registries").WithName("Registry"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Registry")
		os.Exit(1)
	}
	if err = (&controllers.ReplicationReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("registries").WithName("Replication"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Replication")
		os.Exit(1)
	}
	if err = (&controllers.UserReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("registries").WithName("User"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "User")
		os.Exit(1)
	}
	if err = (&controllers.ProjectReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("registries").WithName("Project"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func AddHelmClientReceiver(mgr ctrl.Manager) controllers.HelmClientFactory {
	f := func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		opts := &helmclient.RestConfClientOptions{
			Options: &helmclient.Options{
				Namespace:        namespace,
				RepositoryCache:  repoCache,
				RepositoryConfig: repoConfig,
			},
			RestConfig: mgr.GetConfig(),
		}

		return helmclient.NewClientFromRestConf(opts)
	}
	return f
}
