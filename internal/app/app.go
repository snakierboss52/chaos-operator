package app

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	chaosv1alpha1 "goland-operator/api/v1alpha1"
	"goland-operator/controllers"
	operatorapi "goland-operator/internal/api"
	"goland-operator/internal/infrastructure"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = chaosv1alpha1.AddToScheme(scheme)
}

// Start boots the chaos engineering operator manager
func Start(version string) error {
	banner := `
====================================
  Chaos Engineering Operator
  Version: %s
====================================
`
	fmt.Printf(banner, version)

	// Configure logger
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Get configuration
	metricsAddr := getEnv("METRICS_ADDR", ":8080")
	probeAddr := getEnv("HEALTH_PROBE_ADDR", ":8081")
	apiAddr := getEnv("API_ADDR", ":8082")
	apiAllowedNamespaces := getEnv("API_ALLOWED_NAMESPACES", "chaos-demo")
	enableLeaderElection := getEnv("ENABLE_LEADER_ELECTION", "false") == "true"

	setupLog.Info("Starting manager",
		"version", version,
		"metricsAddr", metricsAddr,
		"probeAddr", probeAddr,
		"apiAddr", apiAddr,
		"apiAllowedNamespaces", apiAllowedNamespaces,
		"leaderElection", enableLeaderElection,
	)

	// Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                ctrl.Options{}.Metrics, // Use default metrics options
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "chaos-operator.chaos.engineering.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	// Create shared result reporter
	reporter := infrastructure.NewResultReporter(mgr.GetClient(), ctrl.Log.WithName("reporter"))

	// Setup PodChaos controller
	if err = (&controllers.PodChaosReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("PodChaos"),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
		Recorder:   mgr.GetEventRecorderFor("podchaos-controller"),
		Reporter:   reporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PodChaos")
		return err
	}

	// Setup StressChaos controller
	if err = (&controllers.StressChaosReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("StressChaos"),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
		Recorder:   mgr.GetEventRecorderFor("stresschaos-controller"),
		Reporter:   reporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StressChaos")
		return err
	}

	// Setup NetworkChaos controller
	if err = (&controllers.NetworkChaosReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("NetworkChaos"),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
		Recorder:   mgr.GetEventRecorderFor("networkchaos-controller"),
		Reporter:   reporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkChaos")
		return err
	}

	// Setup HTTPChaos controller
	if err = (&controllers.HTTPChaosReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("HTTPChaos"),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
		Recorder:   mgr.GetEventRecorderFor("httpchaos-controller"),
		Reporter:   reporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HTTPChaos")
		return err
	}

	// Setup webhooks if enabled
	enableWebhooks := getEnv("ENABLE_WEBHOOKS", "false") == "true"
	if enableWebhooks {
		if err = (&chaosv1alpha1.PodChaos{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "PodChaos")
			return err
		}
		if err = (&chaosv1alpha1.NetworkChaos{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NetworkChaos")
			return err
		}
		if err = (&chaosv1alpha1.StressChaos{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "StressChaos")
			return err
		}
		if err = (&chaosv1alpha1.HTTPChaos{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "HTTPChaos")
			return err
		}
		setupLog.Info("Webhooks enabled")
	}

	// Add health and ready checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}
	if err := mgr.Add(operatorapi.NewServer(
		apiAddr,
		mgr.GetClient(),
		mgr.GetAPIReader(),
		apiAllowedNamespaces,
		ctrl.Log.WithName("http-api"),
	)); err != nil {
		setupLog.Error(err, "unable to add HTTP API server")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
